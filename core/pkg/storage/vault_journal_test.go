package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Тесты дневника (Шаг 10). Всё, что проверяется ниже, — про единственную вещь,
// которую хранилище обещает вызывающему: состояние, возвращённое без ошибки,
// переживает перезапуска процесса, а состояние, до которого дело не дошло, не
// притворяется пережившим.

// TestJournalReplaysFramesAfterReopen — годные кадры поднимаются поверх
// снапшота по одному, в номерном порядке, и возвращают сообщения, диалог и
// поиск в том же виде, в котором их оставили.
func TestJournalReplaysFramesAfterReopen(t *testing.T) {
	db := openDB(t)

	for i, body := range []string{"первое", "второе", "третье"} {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("m%d", i),
			ConversationID: "c1", Body: body, TimestampMS: int64(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	before := db.JournalStats()
	if before.Applied != 0 || before.JournalSeq != 3 || before.Compaction != 0 {
		t.Fatalf("свежий дневник выглядит не так: %+v", before)
	}

	reopened := reopenDB(t, db)
	got := reopened.JournalStats()
	if got.Applied != 3 || got.Skipped != 0 || got.Corrupt != 0 {
		t.Fatalf("переиграно не то: %+v", got)
	}
	if got.JournalSeq != 3 {
		t.Fatalf("номер дневника после открытия другой: %+v", got)
	}
	msgs := reopened.GetMessages("c1", 50)
	// GetMessages отдаёт новые вперёд — так читает UI, и дневник не имеет права
	// менять этот порядок на «как успел переиграться».
	if len(msgs) != 3 || msgs[0].Body != "третье" || msgs[2].Body != "первое" {
		t.Fatalf("порядок сообщений после открытия нарушен: %d %+v", len(msgs), msgs)
	}
	if hits := reopened.SearchMessages("второе", 10); len(hits) != 1 {
		t.Fatalf("полнотекстовый индекс не пережит: %d попаданий", len(hits))
	}
}

// TestJournalCompactAbsorbsFrames — компакция обязана забрать кадры в снапшот
// так, чтобы они не применились вторично, а индексы и поиск остались целыми:
// иначе «дневник догнал снапшот» означало бы удвоенную почту или пропавший
// диалог.
func TestJournalCompactAbsorbsFrames(t *testing.T) {
	db := openDB(t)

	body := strings.Repeat("текст письма ", 200)
	const total = 40
	for i := 0; i < total; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("m%03d", i),
			ConversationID: "c1", Body: body, TimestampMS: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	st := db.JournalStats()
	if st.Compaction == 0 {
		t.Fatalf("за %d кадров не было ни одной компакции: %+v", total, st)
	}
	if st.SnapBytes == 0 {
		t.Fatalf("после компакции снапшота на диске нет: %+v", st)
	}
	if stale, err := filepath.Glob(filepath.Join(db.dataDir, VaultJournalFile+".old-*")); err != nil || len(stale) != 0 {
		t.Fatalf("осиротевшие файлы ротации не убраны: %v (%v)", stale, err)
	}

	reopened := reopenDB(t, db)
	if len(reopened.GetMessages("c1", 1000)) != total {
		t.Fatalf("после компакции в диалоге %d сообщений, ожидалось %d",
			len(reopened.GetMessages("c1", 1000)), total)
	}
	got := reopened.JournalStats()
	if got.Skipped != 0 {
		t.Fatalf("годный дневник после ротации переигран с пропусками: %+v", got)
	}
	if got.Applied >= total || got.JournalSeq != st.JournalSeq {
		t.Fatalf("компакция не закрыла дневник своим номером: было %+v, стало %+v", st, got)
	}
	// Поиск обязан остаться однозначным: вторичное применение кадра
	// пересоздавало бы строки диалога и плодило дубликаты в индексе.
	if hits := reopened.SearchMessages("текст письма", 1000); len(hits) != total {
		t.Fatalf("полнотекстовый индекс даёт %d строк, ожидалось %d", len(hits), total)
	}
	// Дальше дневник пишется заново, с номера снапшота: продолжение диалога не
	// должно возвращаться к старым seq.
	if err := reopened.SaveMessage(&StoredMessage{MessageID: "mпосле", ConversationID: "c1",
		Body: "после компакции"}); err != nil {
		t.Fatal(err)
	}
	if seq := reopened.JournalStats().JournalSeq; seq <= st.JournalSeq {
		t.Fatalf("кадр после компакции получил не возросший номер: %d против %d", seq, st.JournalSeq)
	}
	again := reopenDB(t, reopened)
	if len(again.GetMessages("c1", 1000)) != total+1 {
		t.Fatalf("кадр после компакции потерян: %d сообщений", len(again.GetMessages("c1", 1000)))
	}
	if hits := again.SearchMessages("после компакции", 10); len(hits) != 1 {
		t.Fatalf("новое сообщение не попало в поиск: %d", len(hits))
	}
}

// TestJournalFramesInsideSnapshotAreSkipped — окно аварии между шагом 2
// (снапшот записан) и шагом 3 (дневник повёрнут): кадры, уже включённые в
// снимок, лежат на месте. Открытие обязано пропустить их по номеру, а не
// применить вторично, иначе состояние переписалось бы поверх более свежего.
func TestJournalFramesInsideSnapshotAreSkipped(t *testing.T) {
	db := openDB(t)

	const total = 5
	for i := 0; i < total; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("m%d", i),
			ConversationID: "c1", Body: "в снимке", TimestampMS: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	// Снимаем состояние, не трогая дневник: ровно то, что остаётся после
	// обрыва процесса между записью снапшота и поворотом журнала.
	db.mu.Lock()
	err := db.writeSnapshotLocked()
	db.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openDBAt(t, db.dataDir)
	st := reopened.JournalStats()
	if st.Applied != 0 || st.Skipped != total {
		t.Fatalf("кадры внутри снапшота применились заново: %+v", st)
	}
	if len(reopened.GetMessages("c1", 50)) != total {
		t.Fatalf("пропуск кадров съел сообщения: %d", len(reopened.GetMessages("c1", 50)))
	}
	// Дальнейшая запись продолжается с номера снапшота и не воюет с
	// пропущенными кадрами.
	if err := reopened.SaveMessage(&StoredMessage{MessageID: "mновое", ConversationID: "c1",
		Body: "поверх снимка"}); err != nil {
		t.Fatal(err)
	}
	if got := reopened.JournalStats().JournalSeq; got != total+1 {
		t.Fatalf("новый кадр получил номер %d, ожидалось %d", got, total+1)
	}
	after := reopenDB(t, reopened)
	if len(after.GetMessages("c1", 50)) != total+1 {
		t.Fatalf("запись поверх пропущенных кадров потеряна: %d", len(after.GetMessages("c1", 50)))
	}
}

// TestJournalTornTailTruncated — оборванный на середине кадр — это «запись не
// состоялась», а не мусор в базе: хвост отрезается, считается, годное до него
// остаётся, и дневник после этого пригоден для новых записей.
func TestJournalTornTailTruncated(t *testing.T) {
	db := openDB(t)

	for i := 0; i < 4; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("m%d", i),
			ConversationID: "c1", Body: "целое", TimestampMS: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveSession("session-1", []byte("состояние рэтчета")); err != nil {
		t.Fatal(err)
	}
	good := db.JournalStats().JournalSeq

	path := filepath.Join(db.dataDir, VaultJournalFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Обрывок в середине кадра: заголовок обещает 64 байта, а дописано 20.
	torn := append([]byte(nil), raw...)
	torn = append(torn, 0, 0, 0, 64)
	torn = append(torn, bytes_(20)...)
	if err := os.WriteFile(path, torn, 0600); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openDBAt(t, db.dataDir)
	st := reopened.JournalStats()
	if st.Corrupt == 0 {
		t.Fatalf("битый хвост не замечен: %+v", st)
	}
	if st.JournalSeq != good {
		t.Fatalf("после отрезки хвоста номер дневника уехал: %d вместо %d", st.JournalSeq, good)
	}
	if len(reopened.GetMessages("c1", 50)) != 4 {
		t.Fatalf("целые записи задеты обрезкой: %d сообщений", len(reopened.GetMessages("c1", 50)))
	}
	if blob := reopened.Sessions()["session-1"]; string(blob) != "состояние рэтчета" {
		t.Fatalf("сеанс до обрыва потерян: %q", blob)
	}
	// Файл обязан остаться ровно до границы годного: иначе мусор бы копился.
	now, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(now)) != int64(len(raw)) {
		t.Fatalf("хвост не отрезан: было %d, стало %d байт", len(raw), len(now))
	}
	// Запись после обрезки продолжается по тому же счётчику и не спорит с
	// уцелевшими кадрами.
	if err := reopened.SaveMessage(&StoredMessage{MessageID: "mпосле", ConversationID: "c1",
		Body: "дописано после обрыва"}); err != nil {
		t.Fatal(err)
	}
	again := reopenDB(t, reopened)
	if got := again.GetMessages("c1", 50); len(got) != 5 {
		t.Fatalf("допись после обрыва не пережила перезапуск: %d", len(got))
	}
}

// TestOpenSameDirRefusedWhileHeld — у дневника один владелец на каталог. Второй
// живой экземпляр на тот же каталог писал бы кадры под уже использованными
// номерами, и открытие отбросило бы часть из них как устаревшие: молча потерять
// почту хуже, чем не открыться.
func TestOpenSameDirRefusedWhileHeld(t *testing.T) {
	var key [32]byte
	copy(key[:], "device-master-key-for-tests-0000")
	dir := t.TempDir()

	first, err := OpenEncryptedDB(dir, key, "владелец")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })

	if err := first.SaveMessage(&StoredMessage{MessageID: "m1", ConversationID: "c1", Body: "письмо"}); err != nil {
		t.Fatal(err)
	}
	// Тот же каталог, записанный иначе, — тот же каталог: нормализация пути не
	// даёт «второму владельцу» проскользнуть через слэш или относительную форму.
	for _, alias := range []string{dir + string(filepath.Separator), filepath.Clean(dir) + string(filepath.Separator) + "."} {
		second, err := OpenEncryptedDB(alias, key, "узурпатор")
		if err == nil {
			_ = second.Close()
			t.Fatalf("второй экземпляр открылся без ошибки: %s", alias)
		}
		if second != nil {
			t.Fatalf("при отказе возвращён экземпляр: %s", alias)
		}
		if !strings.Contains(err.Error(), "уже открыт") {
			t.Fatalf("отказ не по причине занятости: %v", err)
		}
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	third, err := OpenEncryptedDB(dir, key, "сменщик")
	if err != nil {
		t.Fatalf("после освобождения каталог не открылся: %v", err)
	}
	t.Cleanup(func() { _ = third.Close() })
	if got := third.GetMessages("c1", 50); len(got) != 1 || got[0].Body != "письмо" {
		t.Fatalf("сменщик не нашёл почту прежнего владельца: %+v", got)
	}
}

// TestClosedVaultRefusesWrites — закрытое хранилище не принимает записей: иначе
// остановленное ядро продолжало бы «успешно» писать в базу, которую никто не
// собирается перечитывать.
func TestClosedVaultRefusesWrites(t *testing.T) {
	db := openDB(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveMessage(&StoredMessage{MessageID: "m1", ConversationID: "c1"}); err == nil {
		t.Fatal("запись в закрытое хранилище принята")
	}
	if err := db.SaveSession("s", []byte("x")); err == nil {
		t.Fatal("сеанс в закрытое хранилище принят")
	}
	// Повторное закрытие — не ошибка: остановка могла произойти и по cleanup.
	if err := db.Close(); err != nil {
		t.Fatalf("повторное закрытие: %v", err)
	}
}

// TestJournalDelSurvivesReopen — удаление из дневника переживает перезапуск.
// Снятое из очереди сообщение не должно возвращаться и уезжать повторно:
// кладка конверта чужому релею считается по счётчику попыток.
func TestJournalDelSurvivesReopen(t *testing.T) {
	db := openDB(t)

	if err := db.QueueOutbound(&StoredOutbound{MessageID: "m1", ConversationID: "c1",
		NodeKey: []byte("0123456789abcdef0123456789abcdef"), Envelope: []byte("кадр"), EnqueuedMS: 10}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueueOutbound(&StoredOutbound{MessageID: "m2", ConversationID: "c1",
		NodeKey: []byte("0123456789abcdef0123456789abcdef"), Envelope: []byte("кадр2"), EnqueuedMS: 20}); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteOutbound("m1"); err != nil {
		t.Fatal(err)
	}

	reopened := reopenDB(t, db)
	pending := reopened.PendingOutbound()
	if len(pending) != 1 || pending[0].MessageID != "m2" {
		t.Fatalf("удаление не пережило перезапуск: %+v", pending)
	}
	if st := reopened.JournalStats(); st.Applied != 3 || st.Skipped != 0 {
		t.Fatalf("кадры очереди переиграны не так: %+v", st)
	}
	// Удаление несуществующего не пишется: кадр «убери m1» вторично — это шум,
	// который при открытии уже ничего не убирает.
	if err := reopened.CompleteOutbound("m-нет-такого"); err != nil {
		t.Fatalf("CompleteOutbound по несуществующей записи: %v", err)
	}
	if seq := reopened.JournalStats().JournalSeq; seq != 3 {
		t.Fatalf("пустое удаление сдвинуло номер дневника: %d", seq)
	}
}

// TestJournalRefusesUnsupportedDel — messages и crdt_ops не удаляют записей, и
// дневник обязан сказать об этом вслух, а не промолчать и оставить в базе
// строку, которую вызывающий считает снятой.
func TestJournalRefusesUnsupportedDel(t *testing.T) {
	db := openDB(t)
	if err := db.SaveMessage(&StoredMessage{MessageID: "m1", ConversationID: "c1"}); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{tableMessages, tableCRDTOps} {
		db.mu.Lock()
		err := db.journalDelLocked(table, "m1")
		db.mu.Unlock()
		if err == nil {
			t.Fatalf("удаление из %s принято дневником", table)
		}
		if !strings.Contains(err.Error(), "не удаляет записей") {
			t.Fatalf("отказ по %s не по причине: %v", table, err)
		}
	}
	// Отказ не оставляет кадра: номер дневника стоит на месте.
	if st := db.JournalStats(); st.JournalSeq != 1 {
		t.Fatalf("отказавшее удаление дописало кадр: %+v", st)
	}
}

// TestJournalEntryKeyMustMatchPayload — кадр, чей ключ не совпадает с
// идентичностью внутри значения, не применяется: иначе подменённый или
// перенесённый кадр переписал бы чужую строку, и никто бы не узнал, что база
// разошлась с тем, что в неё клали.
func TestJournalEntryKeyMustMatchPayload(t *testing.T) {
	db := openDB(t)

	raw, err := json.Marshal(&StoredMessage{MessageID: "другой", ConversationID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	db.mu.Lock()
	mismatch := db.applyEntryLocked(journalEntry{Seq: 1, Op: journalOpSet,
		Table: tableMessages, Key: "первый", Value: raw})
	noSeq := db.applyEntryLocked(journalEntry{Op: journalOpSet, Table: tableMessages,
		Key: "первый", Value: raw})
	unknownTable := db.applyEntryLocked(journalEntry{Seq: 2, Op: journalOpSet,
		Table: "таблицы_такой_нет", Key: "k", Value: raw})
	unknownOp := db.applyEntryLocked(journalEntry{Seq: 3, Op: "перемонтировано",
		Table: tableMessages, Key: "первый", Value: raw})
	db.mu.Unlock()

	if mismatch == nil || !strings.Contains(mismatch.Error(), "не совпадает") {
		t.Fatalf("расхождение ключа и сообщения пропущено: %v", mismatch)
	}
	if noSeq == nil || !strings.Contains(noSeq.Error(), "без номера") {
		t.Fatalf("кадр без номера пропущен: %v", noSeq)
	}
	if unknownTable == nil || !strings.Contains(unknownTable.Error(), "неизвестная таблица") {
		t.Fatalf("чужая таблица пропущена: %v", unknownTable)
	}
	if unknownOp == nil || !strings.Contains(unknownOp.Error(), "неизвестная операция") {
		t.Fatalf("чужая операция пропущена: %v", unknownOp)
	}
	if got := db.GetMessages("c1", 50); len(got) != 0 {
		t.Fatalf("отвергнутые кадры оставили строки в базе: %+v", got)
	}
}

// TestOldVaultOpensWithoutJournal — база, созданная до Шага 10 (тот же файл,
// тот же AAD, без поля journal_seq), обязана открыться без миграции: иначе
// обновление ядра означало бы «диалогов стало меньше».
func TestOldVaultOpensWithoutJournal(t *testing.T) {
	var key [32]byte
	copy(key[:], "device-master-key-for-tests-0000")
	dir := t.TempDir()

	db, err := OpenEncryptedDB(dir, key, "прежняя база")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveMessage(&StoredMessage{MessageID: "m1", ConversationID: "c1",
		Body: "до дневника", TimestampMS: 5}); err != nil {
		t.Fatal(err)
	}
	// Снимок в старом формате: тот же AAD, только без нового поля.
	db.mu.Lock()
	payload, err := json.Marshal(vaultSnapshot{
		Messages: mapToSlice(db.messages),
		CRDTOps:  db.crdtLog,
		Outbound: outboundToSlice(db.outbound),
		Sessions: db.sessions,
		Contacts: db.contacts,
		Sf:       sfToSlice(db.sf),
	})
	if err != nil {
		db.mu.Unlock()
		t.Fatal(err)
	}
	blob, err := db.encryptRecord(payload, snapshotRecordID)
	db.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, VaultSnapshotFile), blob, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, VaultJournalFile)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	legacy := openDBAt(t, dir)
	got := legacy.GetMessages("c1", 50)
	if len(got) != 1 || got[0].Body != "до дневника" {
		t.Fatalf("прежняя база не открылась: %+v", got)
	}
	if st := legacy.JournalStats(); st.JournalSeq != 0 || st.Applied != 0 || st.Skipped != 0 {
		t.Fatalf("пустой дневник старой базы что-то переиграл: %+v", st)
	}
	// Новая запись в такую базу ложится обычным кадром и переживает перезапуск.
	if err := legacy.SaveMessage(&StoredMessage{MessageID: "m2", ConversationID: "c1",
		Body: "после миграции"}); err != nil {
		t.Fatal(err)
	}
	after := reopenDB(t, legacy)
	if got := after.GetMessages("c1", 50); len(got) != 2 {
		t.Fatalf("после записи в прежнюю базу осталось %d сообщений", len(got))
	}
}

func bytes_(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte('a' + i%26)
	}
	return out
}

// BenchmarkJournalAppend — стоимость сохранения одного нового сообщения на
// многотысячной базе: один кадр в дневник.
func BenchmarkJournalAppend(b *testing.B) {
	db := benchPreload(b, benchRows())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("bench-%d", i),
			ConversationID: "bench", Body: "измеряем стоимость записи", TimestampMS: int64(i)}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	st := db.JournalStats()
	b.ReportMetric(float64(st.Compaction), "compactions")
	b.ReportMetric(float64(st.SnapBytes), "snap_bytes")
}

// BenchmarkFullSnapshotRewrite — та же база и то же сообщение, но способом,
// которым запись писалась до Шага 10: полный снимок на каждую запись.
func BenchmarkFullSnapshotRewrite(b *testing.B) {
	db := benchPreload(b, benchRows())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.mu.Lock()
		err := db.writeSnapshotLocked()
		db.mu.Unlock()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(db.JournalStats().SnapBytes), "snap_bytes")
}

// benchRows — размер базы для сравнения «до» и «после»: рядом прогонов с разным
// YGG_BENCH_N видно, растёт ли стоимость записи вместе с размером базы.
func benchRows() int {
	if v := os.Getenv("YGG_BENCH_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3000
}

// benchPreload гоняет базу указанным числом сообщений так, чтобы на диске уже
// лежал снапшот: без него «до» и «после» сравнивали бы пустую базу с
// заполненной.
func benchPreload(b *testing.B, n int) *EncryptedDB {
	b.Helper()
	var key [32]byte
	copy(key[:], "device-master-key-for-tests-0000")
	db, err := OpenEncryptedDB(b.TempDir(), key, "bench")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })
	for i := 0; i < n; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("seed-%06d", i),
			ConversationID: fmt.Sprintf("conv-%d", i%50), Body: strings.Repeat("телеграмма ", 8),
			TimestampMS: int64(i)}); err != nil {
			b.Fatal(err)
		}
	}
	// Снимок обязана оставить компакция: без неё «до» и «после» сравнивали бы
	// пустую базу с заполненной.
	if st := db.JournalStats(); st.Compaction == 0 {
		b.Fatalf("за %d записей не было компакции: %+v", n, st)
	}
	return db
}

// TestForeignJournalIsNotSilentlyEmptied — чужой дневник обязан быть отвергнут, а
// не «обрезан до пустоты». До этого правила открытие каталога с дневником другой
// базы выглядело успехом: первый кадр не вскрывался, хвост считался оборванным,
// файл урезался до нуля, и вызывающий получал пустое состояние без единой ошибки.
// Для бэкапа (core/pkg/rpc/backup_service.go) это означало бы «восстановление
// удалось» при полностью потерянной базе, поэтому отказ обязан быть громким.
func TestForeignJournalIsNotSilentlyEmptied(t *testing.T) {
	var ours, foreign [32]byte
	copy(ours[:], "device-master-key-for-tests-0000")
	copy(foreign[:], "another-device-key-for-tests-001")

	dir := t.TempDir()
	db, err := OpenEncryptedDB(dir, ours, "test-device")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := db.SaveMessage(&StoredMessage{MessageID: fmt.Sprintf("m%d", i),
			ConversationID: "c1", Body: "письмо", TimestampMS: int64(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, VaultJournalFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("дневник не читается: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("дневник пуст: чужой ключ не на чем проверять")
	}

	if other, err := OpenEncryptedDB(dir, foreign, "test-device"); err == nil {
		_ = other.Close()
		t.Fatal("чужой дневник открылся как пустое состояние")
	}
	// Отказ не имеет права править чужой файл: дописи вёл другой владелец.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("дневник после отказа не читается: %v", err)
	}
	if !bytes.Equal(raw, after) {
		t.Fatalf("отказ обрезал чужой дневник: было %d, стало %d байт", len(raw), len(after))
	}

	// Своя база после отвергнутого чужака открывается как ни в чём не бывало.
	back := openDBAt(t, dir)
	if got := back.GetMessages("c1", 50); len(got) != 3 {
		t.Fatalf("после отказа своя база вернула %d сообщений, ожидалось 3", len(got))
	}

	t.Run("невозможная длина первого кадра", func(t *testing.T) {
		garbage := t.TempDir()
		// Длина больше journalFrameMax на нулевом смещении: писатель выдаёт длину и
		// тело одним вызовом, поэтому такого быть не могло даже при обрыве записи.
		blob := []byte{0xFF, 0xFF, 0xFF, 0xFF, 1, 2, 3, 4}
		if err := os.WriteFile(filepath.Join(garbage, VaultJournalFile), blob, 0600); err != nil {
			t.Fatal(err)
		}
		if opened, err := OpenEncryptedDB(garbage, ours, "test-device"); err == nil {
			_ = opened.Close()
			t.Fatal("дневник с невозможной длиной принят")
		}
		if _, err := os.ReadFile(filepath.Join(garbage, VaultJournalFile)); err != nil {
			t.Fatalf("файл после отказа недоступен: %v", err)
		}
	})

	t.Run("оборванный первый кадр переживается", func(t *testing.T) {
		torn := t.TempDir()
		// Ровно первый кадр, обрезанный внутри тела: это исходный случай
		// «запись не состоялась», и он обязан открыться пустой базой без ошибки.
		n := int(binary.BigEndian.Uint32(raw[:4]))
		if n < 8 {
			t.Fatalf("первый кадр длиной %d байт для этого случая не годится", n)
		}
		if err := os.WriteFile(filepath.Join(torn, VaultJournalFile), raw[:4+n-3], 0600); err != nil {
			t.Fatal(err)
		}
		opened, err := OpenEncryptedDB(torn, ours, "test-device")
		if err != nil {
			t.Fatalf("оборванный первый кадр отвергнут: %v", err)
		}
		defer func() { _ = opened.Close() }()
		if st := opened.JournalStats(); st.Corrupt == 0 {
			t.Fatalf("обрезка хвоста не засчитана: %+v", st)
		}
	})
}
