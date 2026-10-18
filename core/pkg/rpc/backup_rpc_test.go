// File: core/pkg/rpc/backup_rpc_test.go
//
// Шаг 13: бэкап вызывается из ядра, поэтому и проверяется как команда FFI, а не
// как библиотека. Все вызовы идут через Engine.Handle — ровно тот путь, по которому
// байты приходят из cmd/libygggram, включая разбор protobuf и заполнение
// CoreResponse_Backup.
//
// Что доказывается этим файлом и почему без этого нельзя:
//
//   - снимок состояния — это файлы хранилища. TestBackupCoversEveryVaultFile
//     перечисляет каталог данных и требует, чтобы каждый файл либо входил в
//     список снимка, либо был назван преходящим. Заведёт pkg/storage третий
//     персистентный файл — тест не пропустит его мимо бэкапа;
//
//   - восстановление на «новом устройстве» существует только вместе с фразой
//     кошелька. TestRestoreRejectsForeignIdentity показывает полезную сторону
//     двух ключей: пакет, снятый с другой идентичности, раскладывается в байты
//     (passphrase при этом верный), но база из этих байтов не открывается, а
//     живая база остаётся нетронутой;
//
//   - события backup_progress — не украшение, а единственный способ UI узнать id
//     пакета и число чанков до расшифровки манифеста. Проверено, что завершающее
//     событие ровно одно, что id в событиях и в ответе совпадают, и что отказ
//     тоже завершает последовательность, а не обрывает её;
//
//   - разбор контейнера — граница недоверия: пакет мог лежать в чужом облаке, а
//     восстановление пишет в каталог данных. Имя вне списка, второй экземпляр
//     того же файла, нарисованная длина и хвост байтов отвергаются до любой
//     записи на диск.
//
//   - цель пакета — это провайдер, а не строка. TestBackupCommandsRefuseBadTargets
//     требует, чтобы незнакомая схема и неподключённая схема отвергались разными
//     текстами, а учётные данные цели не попадали в ответ;
//     TestBackupOverWebDAVThroughEngine гоняет ту же команду поверх настоящего
//     WebDAV-сервера (golang.org/x/net/webdav в этом же процессе): чанки обязаны
//     лечь в хранилище сервера, ответ — назвать их хранилище словом «webdav», а
//     восстановление — поднять базу из сети, а не из локального каталога.
//
// Чего здесь сознательно нет: ни одного обращения к живому облаку (Nextcloud,
// Mail.ru и WebDAV у Yandex.Disk требуют учётных данных, и чем именно они
// отвечают на редирект или на отсутствующий файл, эти тесты не утверждают) и ни
// одной проверки интерфейса — экран бэкапа и мост Dart в этот шаг не входят.
package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/webdav"

	"ygggram/core/pkg/backup"
	"ygggram/core/pkg/storage"
	yggproto "ygggram/core/proto"
)

const backupPassphrase = "ключевая фраза пакета для теста бэкапа"

// runCreate и runRestore зовут команды через Handle: тест обязан идти тем же
// путём, что и живой FFI-вызов, иначе проверенным окажется другой код.
func runCreate(t *testing.T, e *Engine, target, passphrase string) *yggproto.CoreResponse {
	t.Helper()
	return e.Handle(&yggproto.CoreRequest{RequestId: 13, Request: &yggproto.CoreRequest_CreateBackup{
		CreateBackup: &yggproto.CreateBackupReq{
			DestinationTarget:    target,
			EncryptionPassphrase: passphrase,
		},
	}})
}

func runRestore(t *testing.T, e *Engine, target, passphrase string) *yggproto.CoreResponse {
	t.Helper()
	return e.Handle(&yggproto.CoreRequest{RequestId: 14, Request: &yggproto.CoreRequest_RestoreBackup{
		RestoreBackup: &yggproto.RestoreBackupReq{
			SourceTarget:         target,
			EncryptionPassphrase: passphrase,
		},
	}})
}

// dataDirOf — каталог данных, который ядро считает своим. Берётся из поля, а не
// пересобирается из аргумента startIsolated: проверять надо тот путь, по которому
// пойдёт реальная запись.
func dataDirOf(t *testing.T, e *Engine) string {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dataDir == "" {
		t.Fatal("ядро не знает каталога данных")
	}
	return e.dataDir
}

// listDir — имена файлов каталога, без подкаталогов.
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("каталог %s не перечисляется: %v", dir, err)
	}
	out := make([]string, 0, len(entries))
	for _, en := range entries {
		if !en.IsDir() {
			out = append(out, en.Name())
		}
	}
	return out
}

// backupEventsNow вычитывает события, которые уже легли в канал. Прогресс бэкапа
// публикуется синхронно внутри Handle, поэтому к моменту возврата из Handle вся
// последовательность обязана быть на месте — ждать «может, придёт потом» здесь
// нечего. События других веток здесь не нужны и молча пропускаются.
func backupEventsNow(t *testing.T, e *Engine) []*yggproto.BackupProgressEvent {
	t.Helper()
	var out []*yggproto.BackupProgressEvent
	for {
		select {
		case ev := <-e.Events():
			if bp := ev.GetBackupProgress(); bp != nil {
				out = append(out, bp)
			}
		default:
			return out
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func actionsOf(events []*yggproto.BackupProgressEvent) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.GetCurrentAction())
	}
	return out
}

func removeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("%s не удаляется: %v", path, err)
	}
}

// vaultState — состав и содержимое файлов хранилища в каталоге. Сравнивается тем
// же кодом, которым снимок собирается: проверка «ничего не тронуто» обязана
// говорить про те же байты, что читает бэкап, а не про другой список файлов.
func vaultState(t *testing.T, dd string) []vaultFile {
	t.Helper()
	files, err := readVaultFiles(dd)
	if err != nil {
		t.Fatalf("состояние %s не читается: %v", dd, err)
	}
	return files
}

func vaultEqual(a, b []vaultFile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].name != b[i].name || !bytes.Equal(a[i].data, b[i].data) {
			return false
		}
	}
	return true
}

// seedVault поднимает изолированное ядро и добивается, чтобы в базе что-то
// лежало: контакт и два сообщения в очереди. Без записи на диске может и не быть
// ни одного файла, а снимок пустого каталога — не случай для проверки.
//
// Пир берётся из seedBob, поэтому вызывающий обязан передавать ДРУГУЮ фразу:
// узел выводит ключ из mnemonic, и совпавшая фраза дала бы ядру маршрут к самому
// себе — «изолированная» отправка ушла бы в оверлей, и тест проверял бы не очередь.
func seedVault(t *testing.T, dir, mnemonic string) (*Engine, *Engine, *yggproto.PreKeyBundle) {
	t.Helper()

	e := New()
	startIsolated(t, e, dir, mnemonic)
	t.Cleanup(func() {
		e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	})

	peer := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, peer)
	if msg := addContactErr(t, e, cardOf(bundle)); msg != "" {
		t.Fatalf("AddContact: %s", msg)
	}
	for i, body := range []string{"первое", "второе"} {
		sendQueued(t, e, bundle.GetIdentityX25519Pubkey(), &yggproto.ChatMessage{
			MessageId:      fmt.Sprintf("msg-13-%d", i),
			ConversationId: "диалог из бэкапа",
			TimestampMs:    uint64(1700000000000 + i),
			Content:        textContent(body),
		})
	}
	return e, peer, bundle
}

// TestBackupCoversEveryVaultFile — список снимка обязан покрывать весь каталог
// данных. Тест перечисляет файлы после реальной работы ядра и требует, чтобы
// каждый был либо в списке снимка, либо назван здесь преходящим. Новый
// персистентный файл в pkg/storage без правки vaultBackupFiles уронит прогон, а не
// растворится в бэкапе молча.
func TestBackupCoversEveryVaultFile(t *testing.T) {
	dir := t.TempDir()
	e, _, _ := seedVault(t, dir, seedAlice)

	// Рестарт: при закрытии база дописывает и ротирует дневник, поэтому
	// перечислять каталог надо после живого цикла, а не сразу после старта.
	stopIsolated(t, e)
	startIsolated(t, e, dir, seedAlice)

	dd := dataDirOf(t, e)
	covered := 0
	for _, name := range listDir(t, dd) {
		switch {
		case isVaultFileName(name):
			covered++
		case name == storage.VaultSnapshotFile+".tmp":
			// Черновик снапшота: пишется перед атомарной заменой. Это пережиток
			// обрыва, состояние живёт в самом снапшоте под своим именем.
		case strings.HasPrefix(name, storage.VaultJournalFile+".old-"):
			// Ротация дневника: старые кадры уже сложены в снапшот и убираются при
			// следующем открытии (sweepStaleJournals).
		default:
			t.Fatalf("в %s появился файл %q: он не в списке снимка и не назван преходящим — бэкап теряет состояние",
				dd, name)
		}
	}
	if covered == 0 {
		t.Fatalf("в %s нет ни одного файла из списка снимка: проверять покрытие нечем", dd)
	}
	if st := vaultState(t, dd); len(st) == 0 {
		t.Fatal("readVaultFiles вернул пустой список при непустом каталоге")
	}
}

// TestBackupRestoresVaultAfterWipe — сквозной путь: живой пакет, затёртый
// каталог, восстановление, перезапуск. Контакты и очередь обязаны вернуться из
// пакета, а не из памяти процесса: файлы на диске удалены до восстановления.
func TestBackupRestoresVaultAfterWipe(t *testing.T) {
	dir := t.TempDir()
	e, peer, bundle := seedVault(t, dir, seedAlice)
	dd := dataDirOf(t, e)

	before := vaultState(t, dd)
	queueBefore := len(queuedSnapshot(t, e))
	if queueBefore == 0 {
		t.Fatal("очередь пуста: сквозной путь не проверяется")
	}

	pkgDir := filepath.Join(t.TempDir(), "bundle")
	resp := runCreate(t, e, pkgDir, backupPassphrase)
	if !resp.GetSuccess() {
		t.Fatalf("CreateBackup: %s", resp.GetErrorMessage())
	}
	manifest := resp.GetBackup().GetManifest()
	if manifest == nil {
		t.Fatal("CreateBackup прошёл без BackupResp.Manifest — ветка CoreResponse снова пуста")
	}
	if manifest.GetChunkCount() == 0 || uint32(len(manifest.GetChunks())) != manifest.GetChunkCount() {
		t.Fatalf("манифест ответа: обещано %d чанков, перечислено %d",
			manifest.GetChunkCount(), len(manifest.GetChunks()))
	}
	if len(manifest.GetRootMerkleHash()) != 32 || len(manifest.GetArgon2IdSalt()) == 0 {
		t.Fatalf("манифест ответа без корня или без salt: %+v", manifest)
	}
	// Провайдер у чанка — имя того хранилища, куда пакет реально лёг: здесь это
	// каталог на диске, и выдумывать облако, которого не было, ответ не вправе.
	for _, c := range manifest.GetChunks() {
		if c.GetStorageProvider() != backup.ProviderLocal {
			t.Fatalf("чанк %s назван выложенным в %q", c.GetChunkId(), c.GetStorageProvider())
		}
		if len(c.GetSha256Hash()) != 32 {
			t.Fatalf("чанк %s несёт хэш длиной %d", c.GetChunkId(), len(c.GetSha256Hash()))
		}
	}

	// Пакет собран на живом ядре — это и есть рабочий случай: резервную копию
	// делают, не останавливая мессенджер.
	stopIsolated(t, e)
	for _, f := range before {
		removeFile(t, filepath.Join(dd, f.name))
	}
	if _, err := readVaultFiles(dd); err == nil {
		t.Fatal("каталог данных не очищен: проверка восстановления ничего не проверяет")
	}

	restore := runRestore(t, e, pkgDir, backupPassphrase)
	if !restore.GetSuccess() {
		t.Fatalf("RestoreBackup: %s", restore.GetErrorMessage())
	}
	if got := restore.GetBackup().GetManifest().GetBackupId(); got != manifest.GetBackupId() {
		t.Fatalf("ответ восстановления называет пакет %d, а создание — %d", got, manifest.GetBackupId())
	}
	if len(restore.GetBackup().GetManifest().GetChunks()) != 0 {
		t.Fatal("при восстановлении пришёл список чанков: он лежит в зашифрованной части")
	}

	if after := vaultState(t, dd); !vaultEqual(before, after) {
		t.Fatalf("восстановленный каталог не совпал со снятым байт в байт: %d файла против %d",
			len(after), len(before))
	}

	// Финальная проверка идёт через команды ядра: оно обязано открыть
	// восстановленную базу и достать из неё ровно то, что было.
	startIsolated(t, e, dir, seedAlice)
	cards := contactsOf(t, e)
	if len(cards) != 1 {
		t.Fatalf("после восстановления контактов %d, ожидался 1", len(cards))
	}
	if !bytes.Equal(cards[0].GetBundle().GetSignedPrekey(), bundle.GetSignedPrekey()) {
		t.Fatal("восстановлен другой контакт")
	}
	pending := queuedSnapshot(t, e)
	if len(pending) != queueBefore {
		t.Fatalf("в очереди %d записей, до бэкапа было %d", len(pending), queueBefore)
	}
	for _, rec := range pending {
		if !bytes.Equal(rec.NodeKey, peer.node.PublicKey()) {
			t.Fatalf("сообщение %s после восстановления идёт на другой узел: %x", rec.MessageID, rec.NodeKey)
		}
	}
}

// TestRestoreRejectsForeignIdentity — пакет, снятый с другой идентичности, обязан
// быть отвергнут до подмены живой базы. Это и есть полезная сторона двух ключей:
// знание passphrase раскладывает пакет в байты, но не открывает базу, которая
// заширована ключом хранилища от фразы кошелька.
func TestRestoreRejectsForeignIdentity(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "bundle")
	source, _, _ := seedVault(t, t.TempDir(), seedAlice)
	if resp := runCreate(t, source, pkgDir, backupPassphrase); !resp.GetSuccess() {
		t.Fatalf("CreateBackup: %s", resp.GetErrorMessage())
	}

	dirB := t.TempDir()
	// Не seedBob: фраза цели не должна совпадать с фразой тестового пира, иначе
	// узел найдёт маршрут к самому себе и «изолированная» отправка уйдёт в сеть.
	target, _, _ := seedVault(t, dirB, seedCarol)
	dd := dataDirOf(t, target)
	before := vaultState(t, dd)
	stopIsolated(t, target)

	resp := runRestore(t, target, pkgDir, backupPassphrase)
	if resp.GetSuccess() {
		t.Fatal("чужой пакет восстановился в базу другой идентичности")
	}
	if !strings.Contains(resp.GetErrorMessage(), "не открываются ключом этого ядра") {
		t.Fatalf("отказ не про ключ базы: %q", resp.GetErrorMessage())
	}
	if after := vaultState(t, dd); !vaultEqual(before, after) {
		t.Fatal("неудачное восстановление тронуло файлы базы")
	}

	// База жива: перезапуск открывает её и находит прежний контакт.
	startIsolated(t, target, dirB, seedCarol)
	if got := contactsOf(t, target); len(got) != 1 {
		t.Fatalf("после отвергнутого восстановления контактов %d", len(got))
	}
}

// TestRestoreRefusedOnRunningCore — подмена файлов под живым хранилищем
// невозможна: у дневника один владелец на каталог, а приёмный цикл дописывал бы
// кадры поверх уже восстановленного снапшота.
func TestRestoreRefusedOnRunningCore(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "bundle")
	source, _, _ := seedVault(t, t.TempDir(), seedAlice)
	if resp := runCreate(t, source, pkgDir, backupPassphrase); !resp.GetSuccess() {
		t.Fatalf("CreateBackup: %s", resp.GetErrorMessage())
	}

	target, _, _ := seedVault(t, t.TempDir(), seedAlice)
	dd := dataDirOf(t, target)
	before := vaultState(t, dd)

	resp := runRestore(t, target, pkgDir, backupPassphrase)
	if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), "RestoreBackup на живом ядре") {
		t.Fatalf("восстановление на живом ядре не отвергнуто: success=%v %q",
			resp.GetSuccess(), resp.GetErrorMessage())
	}
	if after := vaultState(t, dd); !vaultEqual(before, after) {
		t.Fatal("отказ всё-таки тронул базу")
	}
	if events := backupEventsNow(t, target); len(events) != 0 {
		t.Fatalf("отказ до чтения пакета породил %d событий прогресса", len(events))
	}
}

// TestBackupCommandsRefuseBadTargets — границы команды: облако по имени,
// относительный путь, пустая фраза, вызов до StartNode. Ни один отказ не обязан
// оставлять событие или пакет на диске. Каждое подмножество случая берёт свой
// каталог: два ядра в одном процессе на одном каталоге данных не уживаются, и это
// отдельная проверка (claimVaultDir), а не случайность.
func TestBackupCommandsRefuseBadTargets(t *testing.T) {
	t.Run("до StartNode", func(t *testing.T) {
		e := New()
		target := filepath.Join(t.TempDir(), "out")
		resp := runCreate(t, e, target, backupPassphrase)
		if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), "до StartNode") {
			t.Fatalf("CreateBackup без запущенного ядра: %q", resp.GetErrorMessage())
		}
		if events := backupEventsNow(t, e); len(events) != 0 {
			t.Fatalf("отказ до StartNode опубликовал %d событий", len(events))
		}
		if _, err := os.Stat(target); err == nil {
			t.Fatal("отказ до StartNode оставил каталог пакета")
		}
	})

	t.Run("облачный uri", func(t *testing.T) {
		dir := t.TempDir()
		e := New()
		startIsolated(t, e, dir, seedAlice)
		// Исполняются две схемы: абсолютный путь и webdav+http(s). Всё остальное со
		// «://» обязано быть названным отказом, а не каталогом «https:» в текущей
		// директории: «uri не разобран» и «схема не подключена» — разные причины, и
		// вызывающий обязан получить свою.
		cases := []struct{ target, why string }{
			{"webdav://host/backup", "не знакома"},
			{"https://cloud.example/x", "не знакома"},
			{"s3://bucket/key", "не знакома"},
			{"gdrive://user:S3cr3t@cloud.example/dst", "не подключена"},
			{"lan://192.0.2.7:8443/sync", "не подключена"},
		}
		for _, c := range cases {
			resp := runCreate(t, e, c.target, backupPassphrase)
			if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), c.why) {
				t.Fatalf("%s принято как путь: ожидалось %q, получено %q",
					c.target, c.why, resp.GetErrorMessage())
			}
			// Отказ называет схему и цель, но не секрет из userinfo.
			if strings.Contains(resp.GetErrorMessage(), "S3cr3t") {
				t.Fatalf("отказ повторил пароль цели: %q", resp.GetErrorMessage())
			}
			if events := backupEventsNow(t, e); len(events) != 0 {
				t.Fatalf("%s: отказ опубликовал %d событий", c.target, len(events))
			}
		}
	})

	t.Run("пустой или относительный путь", func(t *testing.T) {
		dir := t.TempDir()
		e := New()
		startIsolated(t, e, dir, seedAlice)
		cases := []struct{ target, why string }{
			{"", "пуст"},
			{"   ", "пуст"},
			{filepath.Join("bundle", "out"), "абсолютным"},
		}
		for _, c := range cases {
			resp := runCreate(t, e, c.target, backupPassphrase)
			if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), c.why) {
				t.Fatalf("%q: ожидался отказ %q, получено %q", c.target, c.why, resp.GetErrorMessage())
			}
		}
	})

	t.Run("без passphrase", func(t *testing.T) {
		dir := t.TempDir()
		e := New()
		startIsolated(t, e, dir, seedAlice)

		resp := runCreate(t, e, filepath.Join(dir, "out"), "")
		if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), "encryption_passphrase") {
			t.Fatalf("CreateBackup без ключа: success=%v %q", resp.GetSuccess(), resp.GetErrorMessage())
		}
		stopIsolated(t, e)
		// Восстановление без ключа отвергается до чтения заголовка: без
		// passphrase ключ не вывести вообще.
		resp = runRestore(t, e, filepath.Join(dir, "out"), "")
		if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), "encryption_passphrase") {
			t.Fatalf("RestoreBackup без ключа: success=%v %q", resp.GetSuccess(), resp.GetErrorMessage())
		}
	})

	t.Run("пакета нет", func(t *testing.T) {
		dir := t.TempDir()
		e := New()
		startIsolated(t, e, dir, seedAlice)
		stopIsolated(t, e)

		missing := filepath.Join(dir, "нет-такого")
		resp := runRestore(t, e, missing, backupPassphrase)
		if resp.GetSuccess() {
			t.Fatal("RestoreBackup из пустого каталога прошёл")
		}
		if !strings.Contains(resp.GetErrorMessage(), "header.json") {
			t.Fatalf("отказ не про заголовок пакета: %q", resp.GetErrorMessage())
		}
	})

	t.Run("чужой ключ пакета", func(t *testing.T) {
		dir := t.TempDir()
		e, _, _ := seedVault(t, dir, seedAlice)
		pkgDir := filepath.Join(t.TempDir(), "bundle")
		if resp := runCreate(t, e, pkgDir, backupPassphrase); !resp.GetSuccess() {
			t.Fatalf("CreateBackup: %s", resp.GetErrorMessage())
		}
		dd := dataDirOf(t, e)
		before := vaultState(t, dd)
		stopIsolated(t, e)

		resp := runRestore(t, e, pkgDir, "другая ключевая фраза пакета")
		if resp.GetSuccess() {
			t.Fatal("пакет восстановился чужой passphrase")
		}
		if after := vaultState(t, dd); !vaultEqual(before, after) {
			t.Fatal("отказ по ключу тронул файлы базы")
		}
		if events := backupEventsNow(t, e); len(events) == 0 || !events[len(events)-1].GetIsFinished() {
			t.Fatalf("отказ по ключу не завершил прогресс: %d событий", len(events))
		}
	})
}

// TestBackupProgressEventsPublished — последовательность backup_progress: один id,
// нарастающий прогресс, ровно одно завершающее событие — и при успехе, и при
// отказе. Интерфейс строит шкалу только на этих данных, поэтому «финала нет»
// означало бы висящий экран, а «два финала» — закрытый и снова открытый прогресс.
func TestBackupProgressEventsPublished(t *testing.T) {
	dir := t.TempDir()
	e, _, _ := seedVault(t, dir, seedAlice)
	pkgDir := filepath.Join(t.TempDir(), "bundle")

	resp := runCreate(t, e, pkgDir, backupPassphrase)
	if !resp.GetSuccess() {
		t.Fatalf("CreateBackup: %s", resp.GetErrorMessage())
	}
	events := backupEventsNow(t, e)
	want := []string{backupActionPacking, backupActionChunksWritten, backupActionSealed}
	got := actionsOf(events)
	if !equalStrings(got, want) {
		t.Fatalf("стадии создания пакета %v, ожидалось %v", got, want)
	}
	id := resp.GetBackup().GetManifest().GetBackupId()
	for i, ev := range events {
		if ev.GetBackupId() != id {
			t.Fatalf("событие %d называет пакет %d, ответ — %d", i, ev.GetBackupId(), id)
		}
		if ev.GetError() != "" {
			t.Fatalf("успешное событие %d с текстом отказа: %q", i, ev.GetError())
		}
		if ev.GetIsFinished() != (i == len(events)-1) {
			t.Fatalf("is_finished у события %d = %v, а финальное должно быть одно и последнее",
				i, ev.GetIsFinished())
		}
	}
	last := events[len(events)-1]
	if last.GetTotalChunks() == 0 || last.GetCompletedChunks() != last.GetTotalChunks() {
		t.Fatalf("финальное событие не доложено: %d из %d", last.GetCompletedChunks(), last.GetTotalChunks())
	}
	if last.GetProgressPercentage() != 100 {
		t.Fatalf("процент в финале %v, ожидался 100", last.GetProgressPercentage())
	}

	// Отказ: пакет читается, манифест не вскрывается. Последовательность
	// обязана завершиться с текстом ошибки, а не оборваться на середине.
	stopIsolated(t, e)
	bad := runRestore(t, e, pkgDir, "другая ключевая фраза пакета")
	if bad.GetSuccess() {
		t.Fatal("восстановление чужой passphrase прошло")
	}
	events = backupEventsNow(t, e)
	want = []string{backupActionChunksRestored, backupActionChunksRestored}
	if got := actionsOf(events); !equalStrings(got, want) {
		t.Fatalf("стадии отказа %v, ожидалось %v", got, want)
	}
	if events[0].GetTotalChunks() == 0 || events[0].GetCompletedChunks() != 0 {
		t.Fatalf("открывающее событие обещает %d из %d: total известен из заголовка до расшифровки",
			events[0].GetCompletedChunks(), events[0].GetTotalChunks())
	}
	failed := events[len(events)-1]
	if !failed.GetIsFinished() || failed.GetError() == "" {
		t.Fatalf("финал отказа: finished=%v error=%q", failed.GetIsFinished(), failed.GetError())
	}
	if failed.GetBackupId() != id {
		t.Fatalf("отказ касается пакета %d, а создание — %d: id разошёлся бы с заголовком",
			failed.GetBackupId(), id)
	}
	// База не тронута: восстановление упало задолго до записи.
	startIsolated(t, e, dir, seedAlice)
	if got := contactsOf(t, e); len(got) != 1 {
		t.Fatalf("после отказа восстановления контактов %d", len(got))
	}
}

// TestVaultPayloadRejectsForeignFiles — разбор контейнера пишет в каталог данных,
// поэтому принимает только файлы хранилища. Пакет мог лежать в чужом облаке: это
// граница недоверия, и проверяется она и отдельно (быстро, без криптографии), и
// сквозь команду — чтобы отказ шёл по живому пути, а не только по юниту.
func TestVaultPayloadRejectsForeignFiles(t *testing.T) {
	snapshot := vaultFile{name: storage.VaultSnapshotFile, data: bytes.Repeat([]byte{0x5A}, 48)}
	journal := vaultFile{name: storage.VaultJournalFile, data: bytes.Repeat([]byte{0xA5}, 16)}
	good := []vaultFile{snapshot, journal}

	t.Run("разбор контейнера", func(t *testing.T) {
		if _, err := decodeVaultPayload(encodeVaultPayload(good)); err != nil {
			t.Fatalf("годный контейнер отвергнут: %v", err)
		}

		_, err := decodeVaultPayload(encodeVaultPayload([]vaultFile{
			snapshot,
			{name: "..\\..\\..\\windows\\system32\\evil.dll", data: []byte("evil")},
		}))
		if err == nil || !strings.Contains(err.Error(), "среди файлов хранилища нет") {
			t.Fatalf("имя вне списка принято: %v", err)
		}

		if _, err := decodeVaultPayload(encodeVaultPayload([]vaultFile{snapshot, snapshot})); err == nil {
			t.Fatal("один и тот же файл принят дважды: состояние пришлось бы выбирать наугад")
		}
		if _, err := decodeVaultPayload(encodeVaultPayload(nil)); err == nil {
			t.Fatal("пустой контейнер принят")
		}
		if _, err := decodeVaultPayload(append(encodeVaultPayload(good), 0x00)); err == nil {
			t.Fatal("хвост байтов после разбора принят")
		}

		// Нарисованная длина обязана быть отвергнута до выделения памяти: иначе
		// заголовок на несколько гигабайт превращается в отказ по памяти.
		raw := encodeVaultPayload([]vaultFile{{name: storage.VaultSnapshotFile, data: []byte("abc")}})
		sizeOff := len(backupPayloadMagic) + 4 + 2 + len(storage.VaultSnapshotFile)
		binary.BigEndian.PutUint64(raw[sizeOff:sizeOff+8], 1<<40)
		if _, err := decodeVaultPayload(raw); err == nil {
			t.Fatal("файл в 1 Тб внутри контейнера на 60 байт принят")
		}

		broken := encodeVaultPayload(good)
		broken[0] = 'X'
		if _, err := decodeVaultPayload(broken); err == nil || !strings.Contains(err.Error(), "преамбул") {
			t.Fatalf("чужая преамбула: %v", err)
		}

		// Снапшот НЕ обязателен: база, у которой вся запись лежит в дневнике, —
		// законное состояние, и пакет без снапшота должен разбираться.
		only, err := decodeVaultPayload(encodeVaultPayload([]vaultFile{journal}))
		if err != nil {
			t.Fatalf("пакет без снапшота отвергнут: %v", err)
		}
		if len(only) != 1 || only[0].name != storage.VaultJournalFile {
			t.Fatalf("разбор вернул не дневник: %+v", only)
		}
	})

	t.Run("сквозь команду", func(t *testing.T) {
		payload := encodeVaultPayload([]vaultFile{
			snapshot,
			{name: "ygggram_evil.txt", data: []byte("запись мимо списка")},
		})
		pkgDir := filepath.Join(t.TempDir(), "bundle")
		if _, err := backup.CreateBundle(backupPassphrase, pkgDir, payload); err != nil {
			t.Fatalf("поддельный пакет не собирается: %v", err)
		}

		dir := t.TempDir()
		target, _, _ := seedVault(t, dir, seedAlice)
		dd := dataDirOf(t, target)
		before := vaultState(t, dd)
		stopIsolated(t, target)

		resp := runRestore(t, target, pkgDir, backupPassphrase)
		if resp.GetSuccess() {
			t.Fatal("восстановление приняло файл, которого нет среди файлов хранилища")
		}
		if !strings.Contains(resp.GetErrorMessage(), "среди файлов хранилища нет") {
			t.Fatalf("отказ не про имя файла: %q", resp.GetErrorMessage())
		}
		if _, err := os.Stat(filepath.Join(dd, "ygggram_evil.txt")); err == nil {
			t.Fatal("чужой файл всё равно лег в каталог данных")
		}
		if after := vaultState(t, dd); !vaultEqual(before, after) {
			t.Fatal("отвергнутый пакет изменил базу")
		}
	})
}

// TestCreateBackupTwiceIsContentStable — второй пакет того же состояния: свой id,
// тот же merkle-корень и те же чанки. Граница честности проходит по числу
// хранилищ, а не по числу пакетов: salt выводится заново, только пока в цели нет
// ни одной читаемой записи, поэтому pkgA и pkgB — два независимых каталога, и
// каждый получает свой подкаталог чанков. Одно и то же состояние в одном каталоге
// сводится в один ключ и не удваивается — это уже не задача вызывающего, а
// проверяется здесь же, в TestCreateBackupIntoSameTargetIsBounded. Дедупликация
// внутри одного пакета (и одного ключа) проверяется в pkg/backup,
// TestDeduplicationLooksAtDiskNotOnlyMemory.
func TestCreateBackupTwiceIsContentStable(t *testing.T) {
	e, _, _ := seedVault(t, t.TempDir(), seedAlice)
	pkgA := filepath.Join(t.TempDir(), "a")
	pkgB := filepath.Join(t.TempDir(), "b")

	first := runCreate(t, e, pkgA, backupPassphrase)
	if !first.GetSuccess() {
		t.Fatalf("первый CreateBackup: %s", first.GetErrorMessage())
	}
	backupEventsNow(t, e)
	m1 := first.GetBackup().GetManifest()
	// На диске ровно столько файлов чанков, сколько обещает манифест ответа:
	// раскладка и ответ не расходятся.
	if chunks := countChunkFiles(t, pkgA); chunks != int(m1.GetChunkCount()) {
		t.Fatalf("в %s лежит %d файлов чанков, а манифест обещает %d", pkgA, chunks, m1.GetChunkCount())
	}

	second := runCreate(t, e, pkgB, backupPassphrase)
	if !second.GetSuccess() {
		t.Fatalf("второй CreateBackup: %s", second.GetErrorMessage())
	}
	m2 := second.GetBackup().GetManifest()
	if m1.GetBackupId() == m2.GetBackupId() {
		t.Fatal("два пакета названы одним id: прогресс и ответ неразличимы")
	}
	if !bytes.Equal(m1.GetRootMerkleHash(), m2.GetRootMerkleHash()) {
		t.Fatal("снимок одного и того же состояния дал разные merkle-корни: сборка недетерминирована")
	}
	if m1.GetChunkCount() != m2.GetChunkCount() || uint32(len(m2.GetChunks())) != m1.GetChunkCount() {
		t.Fatalf("чанков стало %d вместо %d при повторном пакете того же состояния",
			m2.GetChunkCount(), m1.GetChunkCount())
	}
	for i, c := range m1.GetChunks() {
		other := m2.GetChunks()[i]
		if c.GetChunkId() != other.GetChunkId() {
			t.Fatalf("чанк %d во втором пакете назван %s вместо %s", i, other.GetChunkId(), c.GetChunkId())
		}
		if c.GetOffset() != other.GetOffset() || c.GetSize() != other.GetSize() {
			t.Fatalf("чанк %d сместился: %d/%d против %d/%d", i,
				other.GetOffset(), other.GetSize(), c.GetOffset(), c.GetSize())
		}
	}
}

// localTree — все файлы хранилища относительно его корня, через «/», с размерами.
// Смотрим на диск, а не на счётчики ответа: «манифест обещает N чанков» и «в
// хранилище N чанков» должны оставаться двумя разными утверждениями.
func localTree(t *testing.T, root string) map[string]int64 {
	t.Helper()

	out := make(map[string]int64, 16)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("обход %s не удался: %v", root, err)
	}
	return out
}

func namesOf(tree map[string]int64) []string {
	out := make([]string, 0, len(tree))
	for name := range tree {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func countWith(tree map[string]int64, has func(string) bool) int {
	n := 0
	for name := range tree {
		if has(name) {
			n++
		}
	}
	return n
}

// TestCreateBackupIntoSameTargetIsBounded — четыре снимка одного и того же
// состояния в одну и ту же цель. Это то, что Шаг 15 обещает вызывающему и чего
// прежний код не давал: неизменное не выкладывается заново, прежние точки
// восстановимы каждая по своему слоту, а их число ограничено, потому что ротация
// убирает старые записи, не трогая чанки, на которые ещё ссылаются оставшиеся.
//
// Измеряется всё по диску и по ListBundles, а не по счётчикам ответа: отчёт
// может соврать о переиспользовании, а число файлов в хранилище — нет.
func TestCreateBackupIntoSameTargetIsBounded(t *testing.T) {
	e, _, _ := seedVault(t, t.TempDir(), seedAlice)
	pkg := filepath.Join(t.TempDir(), "store")
	provider, err := backup.OpenProvider(pkg)
	if err != nil {
		t.Fatalf("цель %s не открывается провайдером: %v", pkg, err)
	}

	isChunk := func(name string) bool { return strings.HasSuffix(name, ".bin") }
	isSlot := func(name string) bool { return strings.HasPrefix(name, "headers/") }

	first := runCreate(t, e, pkg, backupPassphrase)
	if !first.GetSuccess() {
		t.Fatalf("первый CreateBackup: %s", first.GetErrorMessage())
	}
	m1 := first.GetBackup().GetManifest()
	if status := first.GetBackup().GetStatusMessage(); !strings.Contains(status, "пакет в этом хранилище первый") {
		t.Fatalf("пустое хранилище названо переиспользованным: %s", status)
	}
	tree := localTree(t, pkg)
	if n := countWith(tree, isChunk); n != int(m1.GetChunkCount()) {
		t.Fatalf("чанков на диске %d, манифест обещает %d", n, m1.GetChunkCount())
	}
	// Пустая цель даёт два объекта заголовка: алиас и собственный слот. Одно без
	// другого было бы либо прежним перезаписываемым header.json, либо пакетом,
	// который прежний код не умеет читать вовсе.
	if _, ok := tree["header.json"]; !ok {
		t.Fatalf("алиаса нет в дереве: %v", namesOf(tree))
	}
	if n := countWith(tree, isSlot); n != 1 {
		t.Fatalf("слотов %d, после первого пакета ожидался один: %v", n, namesOf(tree))
	}
	dir := func() string {
		bundles, err := backup.ListBundles(provider)
		if err != nil {
			t.Fatalf("записи после первого пакета не перечисляются: %v", err)
		}
		if len(bundles) != 1 {
			t.Fatalf("записей о пакетах %d, ожидалась одна", len(bundles))
		}
		if bundles[0].ChunkDir == "" {
			t.Fatal("у записи первого пакета нет каталога чанков: слот не прочитан")
		}
		if bundles[0].BackupID != m1.GetBackupId() {
			t.Fatalf("записан пакет %d, а ответ назвал %d", bundles[0].BackupID, m1.GetBackupId())
		}
		return bundles[0].ChunkDir
	}()

	second := runCreate(t, e, pkg, backupPassphrase)
	if !second.GetSuccess() {
		t.Fatalf("второй CreateBackup в ту же цель: %s", second.GetErrorMessage())
	}
	m2 := second.GetBackup().GetManifest()
	if m2.GetBackupId() == m1.GetBackupId() {
		t.Fatal("второй пакет назван тем же id")
	}
	if !bytes.Equal(m1.GetRootMerkleHash(), m2.GetRootMerkleHash()) {
		t.Fatal("тот же снимок состояния дал другой merkle-корень")
	}
	if status := second.GetBackup().GetStatusMessage(); !strings.Contains(status, "уже лежали в хранилище, новых не выложено") {
		t.Fatalf("второй пакет отчитался о выкладке чанков: %s", status)
	}
	grew := localTree(t, pkg)
	if n := countWith(grew, isChunk); n != countWith(tree, isChunk) {
		t.Fatalf("чанков стало %d вместо %d при неизменном содержимом", n, countWith(tree, isChunk))
	}
	if len(grew) != len(tree)+1 {
		t.Fatalf("хранилище выросло на %d объектов, ожидался один слот: было %v, стало %v",
			len(grew)-len(tree), namesOf(tree), namesOf(grew))
	}
	// Прежняя точка на месте и перечислима: именно это до Шага 15 было невозможно,
	// потому что header.json перезаписывался вторым пакетом.
	bundles, err := backup.ListBundles(provider)
	if err != nil {
		t.Fatalf("записи после второго пакета не перечисляются: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("записей %d, ожидалось две: %v", len(bundles), namesOf(grew))
	}
	for _, b := range bundles {
		if b.ChunkDir != dir {
			t.Fatalf("пакет %d лежит в каталоге %s, а первый в %s: дедуп между пакетами не работает",
				b.BackupID, b.ChunkDir, dir)
		}
	}
	if bundles[0].BackupID != m2.GetBackupId() {
		t.Fatalf("свежим назван %d, а записан вторым %d", bundles[0].BackupID, m2.GetBackupId())
	}

	// Третий и четвёртый: записей становится не больше backupKeepBundles, а чанки
	// всё те же. Ротация обязана убрать ровно самый старый слот.
	third := runCreate(t, e, pkg, backupPassphrase)
	if !third.GetSuccess() {
		t.Fatalf("третий CreateBackup: %s", third.GetErrorMessage())
	}
	fourth := runCreate(t, e, pkg, backupPassphrase)
	if !fourth.GetSuccess() {
		t.Fatalf("четвёртый CreateBackup: %s", fourth.GetErrorMessage())
	}
	status := fourth.GetBackup().GetStatusMessage()
	if !strings.Contains(status, "ротация убрала 1 пакет(ов)") {
		t.Fatalf("четвёртый пакет не сообщил об удалённой записи: %s", status)
	}
	m4 := fourth.GetBackup().GetManifest()
	bundles, err = backup.ListBundles(provider)
	if err != nil {
		t.Fatalf("записи после ротации не перечисляются: %v", err)
	}
	if len(bundles) != backupKeepBundles {
		t.Fatalf("записей о пакетах %d, лимит %d: хранилище растёт на каждый повтор",
			len(bundles), backupKeepBundles)
	}
	if bundles[0].BackupID != m4.GetBackupId() {
		t.Fatalf("после ротации свежим назван %d, а последним записан %d", bundles[0].BackupID, m4.GetBackupId())
	}
	for _, b := range bundles[1:] {
		if b.BackupID == m1.GetBackupId() {
			t.Fatal("самый старый пакет пережил ротацию, хотя лимит исчерпан")
		}
	}
	end := localTree(t, pkg)
	if n := countWith(end, isChunk); n != countWith(tree, isChunk) {
		t.Fatalf("чанков стало %d вместо %d: ротация тронула данные, а не только записи",
			n, countWith(tree, isChunk))
	}
	if n := countWith(end, isSlot); n != backupKeepBundles {
		t.Fatalf("слотов %d, лимит %d: %v", n, backupKeepBundles, namesOf(end))
	}
	if size, ok := end["header.json"]; !ok || size == 0 {
		t.Fatalf("алиас после ротации пропал или пуст: %v", namesOf(end))
	}

	// Полезная сторона: то, что лежит в хранилище, — снимок текущего состояния, а
	// не его часть. Байты пакета обязаны сойтись с файлами хранилища ядра.
	files, err := readVaultFiles(dataDirOf(t, e))
	if err != nil {
		t.Fatalf("состояние перед сверкой пакета не читается: %v", err)
	}
	got, err := backup.RestoreBundleFromProvider(backupPassphrase, provider)
	if err != nil {
		t.Fatalf("свежий пакет после ротации не раскладывается: %v", err)
	}
	if !bytes.Equal(got, encodeVaultPayload(files)) {
		t.Fatalf("пакет раскрылся в %d байт, а состояние ядра — в %d", len(got), len(encodeVaultPayload(files)))
	}

	// Ни одна фраза ответа не повторяет passphrase: статусы пишутся по
	// фактическим счётчикам и при этом остаются безопасными для журнала UI.
	for _, resp := range []*yggproto.CoreResponse{first, second, third, fourth} {
		text := resp.GetBackup().GetStatusMessage() + "\x00" + resp.GetErrorMessage()
		if strings.Contains(text, backupPassphrase) {
			t.Fatalf("passphrase попал в текст ответа: %s", text)
		}
	}
}

// TestRestoreJournalOnlyPackageReplacesSnapshot — пакет без снапшота (вся запись
// лежит в дневнике, пока не случилась компакция) обязан восстановить каталог
// ровно своим составом. Прежний снапшот при этом удаляется: оставить его рядом с
// чужим по составу дневником — значит открыть базу состоянием, которого не было
// ни в пакете, ни на устройстве, и оба файла вскрываются одним ключом молча.
func TestRestoreJournalOnlyPackageReplacesSnapshot(t *testing.T) {
	dir := t.TempDir()
	e, _, _ := seedVault(t, dir, seedAlice)
	dd := dataDirOf(t, e)

	live := vaultState(t, dd)
	var journal []byte
	for _, f := range live {
		if f.name == storage.VaultJournalFile {
			journal = f.data
		}
	}
	if len(journal) == 0 {
		t.Fatal("дневника на диске нет: собрать пакет без снапшота нечем")
	}

	pkgDir := filepath.Join(t.TempDir(), "bundle")
	if _, err := backup.CreateBundle(backupPassphrase, pkgDir,
		encodeVaultPayload([]vaultFile{{name: storage.VaultJournalFile, data: journal}})); err != nil {
		t.Fatalf("пакет из одного дневника не собирается: %v", err)
	}

	stopIsolated(t, e)
	// Подменяем каталог на «снапшот есть, дневника нет»: ровно та пара, из которой
	// без удаления могла бы сложиться смесь.
	if err := os.WriteFile(filepath.Join(dd, storage.VaultSnapshotFile), bytes.Repeat([]byte{0x11}, 64), 0600); err != nil {
		t.Fatal(err)
	}
	removeFile(t, filepath.Join(dd, storage.VaultJournalFile))

	resp := runRestore(t, e, pkgDir, backupPassphrase)
	if !resp.GetSuccess() {
		t.Fatalf("RestoreBackup пакета без снапшота: %s", resp.GetErrorMessage())
	}
	restored := vaultState(t, dd)
	if len(restored) != 1 || restored[0].name != storage.VaultJournalFile {
		names := make([]string, 0, len(restored))
		for _, f := range restored {
			names = append(names, f.name)
		}
		t.Fatalf("после восстановления в каталоге %v, ожидался один дневник", names)
	}
	if !bytes.Equal(restored[0].data, journal) {
		t.Fatal("восстановленный дневник не совпадает с пакетом")
	}

	// И рабочее подтверждение: база открывается и содержит то, что было в кадрах.
	startIsolated(t, e, dir, seedAlice)
	if got := contactsOf(t, e); len(got) != 1 {
		t.Fatalf("из восстановленного дневника контактов %d, ожидался 1", len(got))
	}
}

func countChunkFiles(t *testing.T, pkgDir string) int {
	t.Helper()
	total := 0
	err := filepath.WalkDir(pkgDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".bin") {
			total++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("обход %s не удался: %v", pkgDir, err)
	}
	if total == 0 {
		t.Fatalf("в %s нет ни одного чанка", pkgDir)
	}
	return total
}

// startReferenceDAV — эталонный WebDAV-сервер в этом же процессе. Команда ядра
// обязана иметь дело с реализацией протокола, написанной не здесь, а не с
// заглушкой, повторяющей представления клиента.
func startReferenceDAV(t *testing.T) (*httptest.Server, webdav.FileSystem) {
	t.Helper()

	fs := webdav.NewMemFS()
	ts := httptest.NewServer(&webdav.Handler{FileSystem: fs, LockSystem: webdav.NewMemLS()})
	t.Cleanup(ts.Close)
	return ts, fs
}

// davTree — хранилище глазами сервера: имя каждого объекта и его размер. Смотрим
// через файловую систему сервера, а не через тот же клиент, который клал байты,
// иначе «манифест обещает N чанков» и «в хранилище N чанков» были бы одним
// утверждением.
func davTree(t *testing.T, fs webdav.FileSystem, dir string, out map[string]int64) {
	t.Helper()

	f, err := fs.OpenFile(context.Background(), dir, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("коллекция %s не открылась: %v", dir, err)
	}
	defer f.Close()
	infos, err := f.Readdir(-1)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("коллекция %s не перечисляется: %v", dir, err)
	}
	for _, info := range infos {
		name := strings.TrimSuffix(dir, "/") + "/" + info.Name()
		if info.IsDir() {
			davTree(t, fs, name, out)
			continue
		}
		out[name] = info.Size()
	}
}

// TestBackupOverWebDAVThroughEngine — сквозной путь TestBackupRestoresVaultAfterWipe
// поверх сети, а не каталога на диске. Доказывает то, чего локальный случай
// доказать не может: удалённый провайдер подключён к команде ядра, а не только к
// юниту; storage_provider в ответе называет то хранилище, куда чанки реально
// легли; восстановление поднимает базу из сети, когда на диске не осталось
// ничего; и ни один текст ответа не повторяет пароль из userinfo.
func TestBackupOverWebDAVThroughEngine(t *testing.T) {
	ts, fs := startReferenceDAV(t)
	const sub = "/ygggram/2026-10"
	const secret = "S3cr3t pw"
	// net/url требует percent-encoding в userinfo: «сырой» нелатинский пароль цель
	// отвергает ещё до первого запроса.
	target := "webdav+http://backup-user:S3cr3t%20pw@" +
		strings.TrimPrefix(ts.URL, "http://") + sub

	dir := t.TempDir()
	e, peer, bundle := seedVault(t, dir, seedAlice)
	dd := dataDirOf(t, e)
	before := vaultState(t, dd)
	queueBefore := len(queuedSnapshot(t, e))
	if queueBefore == 0 {
		t.Fatal("очередь пуста: сквозной сетевой путь не проверяется")
	}

	resp := runCreate(t, e, target, backupPassphrase)
	if !resp.GetSuccess() {
		t.Fatalf("CreateBackup в webdav: %s", resp.GetErrorMessage())
	}
	manifest := resp.GetBackup().GetManifest()
	if manifest == nil {
		t.Fatal("CreateBackup прошёл без BackupResp.Manifest — ветка CoreResponse снова пуста")
	}
	if manifest.GetChunkCount() == 0 || uint32(len(manifest.GetChunks())) != manifest.GetChunkCount() {
		t.Fatalf("манифест ответа: обещано %d чанков, перечислено %d",
			manifest.GetChunkCount(), len(manifest.GetChunks()))
	}
	// Имя хранилища берётся у того же провайдера, куда лег чанк. «local» здесь
	// означал бы, что ядро само решает, где оно писало.
	for _, c := range manifest.GetChunks() {
		if c.GetStorageProvider() != backup.ProviderWebDAV {
			t.Fatalf("чанк %s назван выложенным в %q, а цель была webdav",
				c.GetChunkId(), c.GetStorageProvider())
		}
	}
	if events := actionsOf(backupEventsNow(t, e)); !equalStrings(events,
		[]string{backupActionPacking, backupActionChunksWritten, backupActionSealed}) {
		t.Fatalf("стадии сетевого пакета %v: сетевой провайдер не должен ни добавлять, ни терять стадию", events)
	}

	// Байты действительно в хранилище сервера, а не только в манифесте ответа.
	tree := map[string]int64{}
	davTree(t, fs, sub, tree)
	bins, aliases, slots := 0, 0, map[string]int64{}
	for name, size := range tree {
		switch {
		case strings.HasSuffix(name, ".bin"):
			if size == 0 {
				t.Fatalf("чанк %s лёг в хранилище пустым", name)
			}
			bins++
		case strings.HasSuffix(name, "/"+backup.HeaderName):
			if size == 0 {
				t.Fatalf("алиас %s лёг в хранилище пустым", name)
			}
			aliases++
		case strings.Contains(name, "/headers/"):
			// Запись пакета (Шаг 15): headers/<ключ>/<id>.json. Именно она делает
			// второй бэкап вторым, а не перезаписью первого.
			if size == 0 {
				t.Fatalf("слот %s лёг в хранилище пустым", name)
			}
			slots[name] = size
		default:
			t.Fatalf("в хранилище лёг %q, которого пакет не обещал", name)
		}
	}
	if bins != int(manifest.GetChunkCount()) {
		t.Fatalf("объектов чанков в хранилище %d, манифест обещает %d", bins, manifest.GetChunkCount())
	}
	if aliases != 1 {
		t.Fatalf("алиасов header.json в хранилище %d, ожидался один", aliases)
	}
	if len(slots) != 1 {
		t.Fatalf("записей пакета в хранилище %d, ожидалась одна: %v", len(slots), slotNames(slots))
	}
	slotPath := slotNames(slots)[0]

	// Ни в одном тексте ответа пароля нет — ни сырого, ни в percent-форме.
	for _, text := range []string{resp.GetBackup().GetStatusMessage(), resp.GetErrorMessage()} {
		if strings.Contains(text, "S3cr3t") {
			t.Fatalf("ответ повторил учётные данные цели: %q", text)
		}
	}
	if status := resp.GetBackup().GetStatusMessage(); !strings.Contains(status, backup.ProviderWebDAV) {
		t.Fatalf("статус не называет то хранилище, куда лег пакет: %q", status)
	}

	// Затирание каталога и восстановление из сети: состояние обязано прийти из
	// WebDAV, а не из памяти процесса.
	stopIsolated(t, e)
	for _, f := range before {
		removeFile(t, filepath.Join(dd, f.name))
	}
	if _, err := readVaultFiles(dd); err == nil {
		t.Fatal("каталог данных не очищен: сетевой путь ничего не проверяет")
	}

	restore := runRestore(t, e, target, backupPassphrase)
	if !restore.GetSuccess() {
		t.Fatalf("RestoreBackup из webdav: %s", restore.GetErrorMessage())
	}
	if got := restore.GetBackup().GetManifest().GetBackupId(); got != manifest.GetBackupId() {
		t.Fatalf("ответ восстановления называет пакет %d, а создание — %d", got, manifest.GetBackupId())
	}
	if len(restore.GetBackup().GetManifest().GetChunks()) != 0 {
		t.Fatal("при восстановлении пришёл список чанков: он лежит в зашифрованной части")
	}
	backupEventsNow(t, e)
	if after := vaultState(t, dd); !vaultEqual(before, after) {
		t.Fatalf("восстановленный из сети каталог не совпал со снятым: %d файла против %d",
			len(after), len(before))
	}

	// Рабочее подтверждение: ядро открывает восстановленную базу и видит то, что
	// было, а очередь сообщений идёт тому же узлу.
	startIsolated(t, e, dir, seedAlice)
	cards := contactsOf(t, e)
	if len(cards) != 1 {
		t.Fatalf("после сетевого восстановления контактов %d, ожидался 1", len(cards))
	}
	if !bytes.Equal(cards[0].GetBundle().GetSignedPrekey(), bundle.GetSignedPrekey()) {
		t.Fatal("восстановлен другой контакт")
	}
	pending := queuedSnapshot(t, e)
	if len(pending) != queueBefore {
		t.Fatalf("в очереди %d записей, до пакета было %d", len(pending), queueBefore)
	}
	for _, rec := range pending {
		if !bytes.Equal(rec.NodeKey, peer.node.PublicKey()) {
			t.Fatalf("сообщение %s после восстановления идёт на другой узел: %x", rec.MessageID, rec.NodeKey)
		}
	}

	// Слот переживает потерю алиаса: header.json — указатель, а запись о пакете
	// живёт в headers/<ключ>/<id>.json. Убрав алиас, восстановление обязано найти
	// пакет по слоту: это и есть то, ради чего Шаг 15 ввёл второй объект.
	if err := fs.RemoveAll(context.Background(), sub+"/"+backup.HeaderName); err != nil {
		t.Fatalf("алиас не убрать из хранилища: %v", err)
	}
	recreated := vaultState(t, dd)
	stopIsolated(t, e)
	viaSlot := runRestore(t, e, target, backupPassphrase)
	if !viaSlot.GetSuccess() {
		t.Fatalf("RestoreBackup по одному слоту: %s", viaSlot.GetErrorMessage())
	}
	if got := viaSlot.GetBackup().GetManifest().GetBackupId(); got != manifest.GetBackupId() {
		t.Fatalf("пакет по слоту назван %d, а создан %d", got, manifest.GetBackupId())
	}
	backupEventsNow(t, e)

	// Отказ на сетевом пути называет причину и цель редагированно: чанки целы, а
	// записей о пакете нет ни одной — восстановление обязано сказать «пакета нет»,
	// а не молча открыть пустую базу.
	for _, f := range recreated {
		removeFile(t, filepath.Join(dd, f.name))
	}
	if err := fs.RemoveAll(context.Background(), slotPath); err != nil {
		t.Fatalf("слот не убрать из хранилища: %v", err)
	}
	broken := runRestore(t, e, target, backupPassphrase)
	if broken.GetSuccess() {
		t.Fatal("восстановление без единой записи о пакете прошло")
	}
	if !strings.Contains(broken.GetErrorMessage(), "xxxxxx") {
		t.Fatalf("отказ не редактирует цель: %q", broken.GetErrorMessage())
	}
	if strings.Contains(broken.GetErrorMessage(), "S3cr3t") {
		t.Fatalf("отказ повторил пароль: %q", broken.GetErrorMessage())
	}
	backupEventsNow(t, e)
	if _, err := readVaultFiles(dd); err == nil {
		t.Fatal("отказ до чтения пакета оставил файлы в каталоге данных")
	}
	// Чанки при этом целы: отсутствие записей — не повод хранилищу что-то терять.
	remains := map[string]int64{}
	davTree(t, fs, sub, remains)
	restBins := 0
	for name := range remains {
		if strings.HasSuffix(name, ".bin") {
			restBins++
		}
	}
	if restBins != int(manifest.GetChunkCount()) {
		t.Fatalf("после отказа чанков в хранилище %d, было %d: потеря записи не имеет права стирать данные",
			restBins, manifest.GetChunkCount())
	}
}

func slotNames(slots map[string]int64) []string {
	out := make([]string, 0, len(slots))
	for name := range slots {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
