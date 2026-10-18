// File: core/pkg/rpc/backup_service.go
//
// Шаг 13: у pkg/backup появляется вызывающий внутри ядра. До этого команды
// CreateBackup и RestoreBackup были названными отказами, ветка ответа
// CoreResponse_Backup не наполнялась никогда, а событие backup_progress никто
// не строил — всё три факта держались на реестрах из contract_test.go, и ровно
// туда они и вернутся, если этот файл перестанет их подтверждать.
//
// Что здесь решено и почему:
//
//   - Снимок состояния — это файлы хранилища (ygggram_vault.bin и
//     ygggram_vault.jrl), а не «байты из памяти». Других персистентных файлов
//     ядро не пишет: это проверяет TestBackupCoversEveryVaultFile, и если
//     pkg/storage заведёт третий файл, тест не пропустит его мимо бэкапа.
//   - Фраза восстановления в пакет НЕ кладётся. Ядро её не хранит вовсе —
//     мастер-ключи выводятся из NodeConfig.mnemonic_seed при каждом старте
//     (startNode), — поэтому восстановление на «новом устройстве» означает
//     «та же фраза кошелька + остановленное ядро», а passphrase пакета —
//     второй, независимый ключ. Отсюда полезное свойство: знание passphrase без
//     фразы кошелька даёт пакет, который раскладывается в байты, но база из этих
//     байтов не открывается (проверяет TestRestoreRejectsForeignIdentity).
//   - Создавать пакет можно на живом ядре. Handle держит e.mu, а оба писателя
//     хранилища (receiveLoop и outboxLoop) берут тот же e.mu, поэтому между
//     чтением снапшота и чтением дневника не может состояться ни компакция, ни
//     ротация. Согласованность держится на этом, а не на удаче.
//   - Восстанавливать — только на остановленном: у дневника один владелец на
//     каталог (claimVaultDir), а приёмный цикл дописывал бы кадры поверх
//     уже подменённого снапшота.
//   - Цель пакета разбирает backup.OpenProvider: абсолютный путь в файловой
//     системе и webdav+http(s)://… (Шаг 14). Обе схемы исполняются настоящим
//     кодом, поэтому в ответе storage_provider стоит то имя, куда чанки реально
//     легли. Схемы gdrive/yandex/lan отвергаются по названию: их клиенты не
//     сверялись ни с одним живым API (см. пакетное описание pkg/backup).
//   - Любая сетевая работа идёт внутри Handle, то есть под e.mu, поэтому у
//     каждого запроса webdav есть потолок defaultProviderTimeout. Без него
//     зависшее облако остановило бы приём почты на всём ядре.
package rpc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ygggram/core/pkg/backup"
	"ygggram/core/pkg/storage"
	yggproto "ygggram/core/proto"
)

// vaultBackupFiles — файлы хранилища, из которых собирается снимок. Порядок здесь
// же задаёт порядок сборки пакета: и снапшот, и дневник могут отсутствовать по
// отдельности (в базу ещё не писали, наоборот — запись амортизируется в дневник и
// до компакции снапшота нет), поэтому годится любой непустой набор, а вот
// подмена порядка означала бы, что восстановление кладёт дневник раньше
// снапшота.
var vaultBackupFiles = []string{storage.VaultSnapshotFile, storage.VaultJournalFile}

const (
	// backupPayloadMagic — магическая преамбула контейнера полезной нагрузки:
	// pkg/backup знает только «байты → чанки», а именованные файлы внутри этих
	// байтов живут здесь.
	backupPayloadMagic = "YGGPL1"
	// backupPayloadMaxFiles — за этим числом идёт либо чужой файл, либо
	// испорченный заголовок: файлов хранилища два, и потолок с запасом.
	backupPayloadMaxFiles = 16

	// backupKeepBundles — сколько последних пакетов ротация оставляет в
	// хранилище после каждого CreateBackup. Три — чтобы одна неудачная запись не
	// означала «последняя точка восстановления испорчена», и чтобы хранилище не
	// росло на каждый запуск ядра. Значение выбрано вызывающим, а не движком:
	// pkg/backup умеет чистить по любой границе, а решать, сколько истории нужно
	// пользователю, здесь.
	backupKeepBundles = 3
)

// Стадии backup_progress. current_action machine-friendly (нижний регистр с
// подчёркиванием), человек читает status_message ответа.
const (
	backupActionPacking        = "packing_vault"
	backupActionChunksWritten  = "chunks_written"
	backupActionSealed         = "bundle_sealed"
	backupActionChunksRestored = "chunks_restored"
	backupActionVaultVerified  = "vault_verified"
	backupActionRestored       = "vault_restored"
)

// vaultFile — один файл хранилища внутри контейнера.
type vaultFile struct {
	name string
	data []byte
}

// emitBackupProgress публикует одну стадию бэкапа. e.mu уже удерживается
// вызывающим (Handle), поэтому nextEventID здесь безопасен, а emit не
// блокируется: читатель в UI может быть занят.
func (e *Engine) emitBackupProgress(backupID uint64, completed, total uint32, action string, finished bool, failure string) {
	var percentage float32
	if total > 0 {
		percentage = float32(completed) * 100 / float32(total)
	}
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_BackupProgress{BackupProgress: &yggproto.BackupProgressEvent{
			BackupId:           backupID,
			CompletedChunks:    completed,
			TotalChunks:        total,
			ProgressPercentage: percentage,
			CurrentAction:      action,
			IsFinished:         finished,
			Error:              failure,
		}},
	})
}

// backupErrorText — поле error события: пустое при успехе.
func backupErrorText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// backupTargetProvider разбирает цель пакета и возвращает провайдер. Ошибки
// называют поле запроса, чтобы вызывающий понял, какое из двух сообщений
// исправишь, и никогда не называют пароль из userinfo (Backup.RedactedTarget).
func backupTargetProvider(raw, field string) (backup.CloudProvider, error) {
	provider, err := backup.OpenProvider(raw)
	if err != nil {
		return nil, fmt.Errorf("rpc: %s = %q: %w", field, backup.RedactedTarget(raw), err)
	}
	return provider, nil
}

// readVaultFiles собирает полезные байты снимка. Дневник может не существовать
// (в базу ещё не писали), а снапшот может отсутствовать: запись амортизируется в
// дневник, и свежая база законно живёт одними кадрами. Поэтому требуется хотя бы
// один файл, а не конкретный.
func readVaultFiles(dataDir string) ([]vaultFile, error) {
	out := make([]vaultFile, 0, len(vaultBackupFiles))
	for _, name := range vaultBackupFiles {
		data, err := os.ReadFile(filepath.Join(dataDir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("rpc: файл хранилища %s не читается: %w", name, err)
		}
		out = append(out, vaultFile{name: name, data: data})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("rpc: в %s нет ни %s, ни %s: ядро ещё ничего не записало, бэкапать нечего",
			dataDir, storage.VaultSnapshotFile, storage.VaultJournalFile)
	}
	return out, nil
}

// encodeVaultPayload — контейнер именованных файлов. Размеры фиксированные и в
// big-endian: разбор не зависит от варинтов и не выделяет память по нарисованной
// длине.
func encodeVaultPayload(files []vaultFile) []byte {
	buf := bytes.NewBufferString(backupPayloadMagic)
	var num [8]byte
	binary.BigEndian.PutUint32(num[:4], uint32(len(files)))
	buf.Write(num[:4])
	for _, f := range files {
		binary.BigEndian.PutUint16(num[:2], uint16(len(f.name)))
		buf.Write(num[:2])
		buf.WriteString(f.name)
		binary.BigEndian.PutUint64(num[:], uint64(len(f.data)))
		buf.Write(num[:])
		buf.Write(f.data)
	}
	return buf.Bytes()
}

// decodeVaultPayload разбирает контейнер и не принимает ничего, кроме файлов
// хранилища. Это граница недоверия: пакет мог лежать в чужом облаке, и запись
// произвольного имени в каталог данных означала бы произвольную запись на диск
// (включая «..\\..\\» в имени).
func decodeVaultPayload(raw []byte) ([]vaultFile, error) {
	if len(raw) < len(backupPayloadMagic)+4 {
		return nil, errors.New("rpc: полезная нагрузка пакета короче заголовка контейнера")
	}
	if string(raw[:len(backupPayloadMagic)]) != backupPayloadMagic {
		return nil, fmt.Errorf("rpc: контейнер с преамбулой %q этот код не читает", raw[:len(backupPayloadMagic)])
	}
	off := len(backupPayloadMagic)
	count := binary.BigEndian.Uint32(raw[off : off+4])
	off += 4
	if count == 0 || count > backupPayloadMaxFiles {
		return nil, fmt.Errorf("rpc: контейнер объявляет %d файлов, а их число обязано быть от 1 до %d", count, backupPayloadMaxFiles)
	}

	files := make([]vaultFile, 0, count)
	for i := uint32(0); i < count; i++ {
		if len(raw)-off < 2 {
			return nil, errors.New("rpc: контейнер оборвался на длине имени файла")
		}
		nameLen := int(binary.BigEndian.Uint16(raw[off : off+2]))
		off += 2
		if nameLen == 0 || nameLen > len(raw)-off {
			return nil, fmt.Errorf("rpc: длина имени файла %d не влезает в контейнер", nameLen)
		}
		name := string(raw[off : off+nameLen])
		off += nameLen

		if len(raw)-off < 8 {
			return nil, errors.New("rpc: контейнер оборвался на длине файла")
		}
		size := binary.BigEndian.Uint64(raw[off : off+8])
		off += 8
		// Сравнение с остатком ДО выделения памяти: нарисованная длина в
		// несколько гигабайт не должна превращаться в попытку её зарезервировать.
		if size > uint64(len(raw)-off) {
			return nil, fmt.Errorf("rpc: файл %s объявлен в %d байт, а в контейнере осталось %d", name, size, len(raw)-off)
		}
		data := make([]byte, size)
		copy(data, raw[off:off+int(size)])
		off += int(size)

		if !isVaultFileName(name) {
			return nil, fmt.Errorf("rpc: пакет несёт файл %q, которого среди файлов хранилища нет: восстановление записало бы его в каталог данных", name)
		}
		for _, prev := range files {
			if prev.name == name {
				return nil, fmt.Errorf("rpc: файл %s встречается в контейнере дважды: состояние пришлось бы выбирать наугад", name)
			}
		}
		files = append(files, vaultFile{name: name, data: data})
	}
	if off != len(raw) {
		return nil, fmt.Errorf("rpc: после %d разобранных файлов в контейнере осталось %d байт", len(files), len(raw)-off)
	}
	return files, nil
}

func isVaultFileName(name string) bool {
	for _, known := range vaultBackupFiles {
		if name == known {
			return true
		}
	}
	return false
}

// backupManifestProto — единственный сборщик protobuf-сообщения BackupManifest:
// до Шага 13 его не строил ни один Go-код, хотя поля header.json и ygggram.proto
// названы одинаково. descriptors известны только тому, кто их создавал (список
// чанков лежит в зашифрованной части пакета), поэтому при восстановлении
// приходит пустой список — это отражено в status_message, а не скрыто.
// providerName вызывающий берёт у того же провайдера, куда легли чанки: поле
// storage_provider обязано называть реальное хранилище, а не «local» по умолчанию.
func backupManifestProto(h *backup.Header, descriptors []backup.ChunkDescriptor, providerName string) (*yggproto.BackupManifest, error) {
	manifest := &yggproto.BackupManifest{
		BackupId:        h.BackupID,
		CreatedAt:       uint64(h.CreatedAt),
		TotalBytes:      h.TotalPlain,
		ChunkCount:      h.TotalChunks,
		RootMerkleHash:  append([]byte(nil), h.RootMerkleHash...),
		Argon2IdSalt:    append([]byte(nil), h.Salt...),
		CompressionAlgo: h.Compression,
		EncryptionAlgo:  h.Encryption,
	}
	var offset uint64
	for _, d := range descriptors {
		raw, err := hex.DecodeString(d.ChunkHash)
		if err != nil {
			return nil, fmt.Errorf("rpc: хэш чанка %q не разбирается: %w", d.ChunkHash, err)
		}
		manifest.Chunks = append(manifest.Chunks, &yggproto.BackupChunk{
			ChunkId:         d.ChunkHash,
			Offset:          offset,
			Size:            d.OriginalSize,
			Sha256Hash:      raw,
			StorageProvider: providerName,
		})
		offset += uint64(d.OriginalSize)
	}
	if len(descriptors) > 0 && uint32(len(descriptors)) != h.TotalChunks {
		return nil, fmt.Errorf("rpc: перечислено %d чанков, а заголовок обещает %d", len(descriptors), h.TotalChunks)
	}
	return manifest, nil
}

// createBackup исполняет CoreRequest_CreateBackup.
//
// Именованный return нужен ради ровно одного завершающего события: defer
// публикует финальную стадию на любом выходе, включая ошибку, и это проверяет
// TestBackupProgressEventsPublished. Поэтому внутри err присваивается только
// через «=» и только на верхнем уровне функции: присваивание в блоке if создало
// бы тень, и ошибка ушла бы мимо defer.
func (e *Engine) createBackup(req *yggproto.CreateBackupReq, out *yggproto.BackupResp) (err error) {
	provider, err := backupTargetProvider(req.GetDestinationTarget(), "destination_target")
	if err != nil {
		return err
	}
	target := backup.RedactedTarget(req.GetDestinationTarget())
	passphrase := req.GetEncryptionPassphrase()
	if passphrase == "" {
		return errors.New("rpc: CreateBackup без encryption_passphrase: пакет без ключа не шифруется")
	}
	if e.dataDir == "" {
		return errors.New("rpc: CreateBackup до StartNode: каталог данных ещё не известен, снимать нечего")
	}

	files, err := readVaultFiles(e.dataDir)
	if err != nil {
		return err
	}
	payload := encodeVaultPayload(files)

	// Дедупликация и разговор с хранилищем начинаются внутри конструктора:
	// провайдер перечисляет свои чанки до того, как уехал первый байт. Ключ при
	// этом берётся у самого свежего пакета хранилища, поэтому второй снимок
	// ложится в тот же каталог чанков и не выкладывает неизменное заново.
	engine, err := backup.NewSnapshotEngineToProvider(passphrase, provider)
	if err != nil {
		return fmt.Errorf("rpc: пакет в %s не заводится: %w", target, err)
	}
	backupID := engine.BackupID()
	var completed, total uint32
	action := backupActionPacking
	defer func() {
		e.emitBackupProgress(backupID, completed, total, action, true, backupErrorText(err))
	}()
	e.emitBackupProgress(backupID, 0, 0, backupActionPacking, false, "")

	descriptors, totalChunkBytes, err := engine.ProcessFileIntoChunks(payload)
	if err != nil {
		return fmt.Errorf("rpc: раскладка чанков не удалась: %w", err)
	}
	completed = uint32(len(descriptors))
	total = completed
	action = backupActionChunksWritten
	e.emitBackupProgress(backupID, completed, total, backupActionChunksWritten, false, "")

	header, err := engine.Seal(descriptors, uint64(len(payload)), totalChunkBytes)
	if err != nil {
		return fmt.Errorf("rpc: манифест не собрался: %w", err)
	}
	// Заголовок выкладывается последним: чанки обязаны существовать раньше, чем
	// появится файл, по которому их начинают искать. WriteBundle кладёт его дважды
	// — в собственный слот и в алиас header.json, — поэтому прежняя точка
	// восстановления перестаёт быть перезаписанной.
	if err = engine.WriteBundle(header); err != nil {
		return fmt.Errorf("rpc: заголовок пакета не выложен в %s: %w", target, err)
	}

	manifest, err := backupManifestProto(header, descriptors, provider.Name())
	if err != nil {
		return err
	}
	out.Manifest = manifest

	stale, pruneErr := backup.PruneBundles(passphrase, provider, backupKeepBundles)
	uploaded, alreadyThere := engine.DedupStats()
	out.StatusMessage = createBackupStatus(provider.Name(), header, engine.ReusedKey(),
		uploaded, alreadyThere, stale, pruneErr)
	if pruneErr != nil {
		// Пакет создан и читается, поэтому отказ ротации не отменяет бэкап: он
		// называется в status_message. Оставлять его молча нельзя — вызывающий
		// вправе знать, что хранилище не очистилось и повтор захода нужен.
		out.StatusMessage += fmt.Sprintf("; ротация не завершилась: %s (пакет цел, повторите создание)", pruneErr)
	}
	action = backupActionSealed
	return nil
}

// createBackupStatus — человекочитаемый итог создания пакета. Провайдер обязан
// быть назван тем именем, куда легли чанки (это проверяет
// TestBackupOverWebDAVThroughEngine), а про дедупликацию и ротацию говорится по
// фактическим счётчикам, а не по названию команды. Ни passphrase, ни учётные
// данные цели в текст не попадают.
func createBackupStatus(providerName string, h *backup.Header, reused bool, uploaded, reusedChunks uint32, rep backup.PruneReport, pruneErr error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "снимок %d байт уложен в %d чанк(ов) в хранилище %s",
		h.TotalPlain, h.TotalChunks, providerName)
	switch {
	case !reused:
		b.WriteString("; пакет в этом хранилище первый: salt выведен заново, переиспользовать чанки не из чего")
	case reusedChunks == h.TotalChunks:
		fmt.Fprintf(&b, "; ключ и каталог чанков взяты у самого свежего пакета, все %d чанк(ов) уже лежали в хранилище, новых не выложено", reusedChunks)
	default:
		fmt.Fprintf(&b, "; ключ и каталог чанков взяты у самого свежего пакета: %d чанк(ов) уже лежали в хранилище, %d выложено новь",
			reusedChunks, uploaded)
	}
	switch {
	case pruneErr != nil: // причину дописывает вызывающий: пакет от этого не становится хуже
	case rep.Dropped > 0:
		fmt.Fprintf(&b, "; ротация убрала %d пакет(ов) и %d чанк(ов) (%d байт), записей осталось %d из %d",
			rep.Dropped, rep.ChunksDeleted, rep.BytesFreed, rep.Kept, rep.Bundles)
	case len(rep.Skipped) > 0:
		fmt.Fprintf(&b, "; ротация пропустила %d пакет(ов) по названным причинам, записей %d из %d",
			len(rep.Skipped), rep.Kept, rep.Bundles)
	default:
		fmt.Fprintf(&b, "; в хранилище %d пакет(ов), удалять нечего", rep.Bundles)
	}
	return b.String()
}

// restoreBackup исполняет CoreRequest_RestoreBackup. Порядок: развернуть пакет в
// память, проверить, что восстановленная база открывается ключом ЭТОГО ядра, и
// только потом трогать каталог данных.
func (e *Engine) restoreBackup(req *yggproto.RestoreBackupReq, out *yggproto.BackupResp) (err error) {
	provider, err := backupTargetProvider(req.GetSourceTarget(), "source_target")
	if err != nil {
		return err
	}
	target := backup.RedactedTarget(req.GetSourceTarget())
	passphrase := req.GetEncryptionPassphrase()
	if passphrase == "" {
		return errors.New("rpc: RestoreBackup без encryption_passphrase: ключ выводится из неё, а не угадывается")
	}
	if e.running {
		return errors.New("rpc: RestoreBackup на живом ядре: у дневника обязан быть один владелец каталога, а приёмный цикл дописывал бы кадры поверх подменённого снапшота — сначала StopNode")
	}
	if e.dataDir == "" || e.vaultKey == nil || e.device == nil {
		return errors.New("rpc: RestoreBackup без успешного StartNode: ядро не знает ни куда восстанавливать, ни каким ключом открывать базу")
	}
	// Старт, оборвавшийся уже после открытия базы: running false, а дневник
	// каталога занят. Подменять файлы под открытым хранилищем нельзя и с
	// погасшим приёмным циклом — закрытое и записанное на закрытии состояние
	// перемешается с восстановленным.
	if e.db != nil {
		return errors.New("rpc: RestoreBackup при незакрытом хранилище: предыдущий StartNode оборвался после открытия базы, нужен StopNode")
	}

	// Самый свежий пакет, а не «файл с фиксированным именем»: с Шага 15 записей о
	// пакетах несколько, и восстановление по умолчанию обязано читать последнюю
	// точку, а не ту, которую листинг поставил первым.
	newest, err := backup.NewestBundle(provider)
	if err != nil {
		return fmt.Errorf("rpc: пакет в %s не читается: %w", target, err)
	}
	header := newest.Header
	backupID := header.BackupID
	total := header.TotalChunks
	var completed uint32
	action := backupActionChunksRestored
	defer func() {
		e.emitBackupProgress(backupID, completed, total, action, true, backupErrorText(err))
	}()
	// Открытие с настоящим total: число чанков известно из заголовка ещё до
	// расшифровки манифеста, и UI вправе нарисовать шкалу сразу. По мере сборки
	// почанковых чисел не будет — движок восстановления не умеет докладывать
	// внутри Restore, поэтому следующий отчёт придёт уже готовым.
	e.emitBackupProgress(backupID, 0, total, backupActionChunksRestored, false, "")

	engine, err := backup.NewSnapshotEngineWithProvider(passphrase, provider, header.Salt, backup.KDFParams{
		Time:      header.Time,
		MemoryKiB: header.MemoryKiB,
		Threads:   header.Threads,
	})
	if err != nil {
		return fmt.Errorf("rpc: ключ из passphrase не выводится: %w", err)
	}
	payload, err := engine.Restore(header)
	if err != nil {
		return fmt.Errorf("rpc: пакет не восстанавливается: %w", err)
	}
	completed = total
	e.emitBackupProgress(backupID, completed, total, backupActionChunksRestored, false, "")

	files, err := decodeVaultPayload(payload)
	if err != nil {
		return err
	}

	// Пробное открытие в отдельном временном каталоге: живые файлы трогать
	// нельзя, пока не ясно, что база вообще открывается этим ключом.
	tmp, err := os.MkdirTemp("", "ygggram-restore-")
	if err != nil {
		return fmt.Errorf("rpc: временный каталог проверки не создать: %w", err)
	}
	defer os.RemoveAll(tmp)
	for _, f := range files {
		if err = os.WriteFile(filepath.Join(tmp, f.name), f.data, 0600); err != nil {
			return fmt.Errorf("rpc: проверка восстановления: %s не записан: %w", f.name, err)
		}
	}
	action = backupActionVaultVerified
	probe, err := storage.OpenEncryptedDB(tmp, *e.vaultKey, e.device.Certificate.DeviceID)
	if err != nil {
		return fmt.Errorf("rpc: восстановленные файлы не открываются ключом этого ядра (база не тронута): %w — пакет снят с другой идентичности или другим master_key_256", err)
	}
	if err = probe.Close(); err != nil {
		return fmt.Errorf("rpc: проверочная база не закрылась: %w", err)
	}
	e.emitBackupProgress(backupID, completed, total, backupActionVaultVerified, false, "")

	// Замена: сначала убрать ОБА файла хранилища, потом класть то, что несёт
	// пакет. Порядок важен не ради атомарности, а ради состава: пакет может быть
	// снят с базы, у которой снапшота ещё нет (запись амортизируется в дневник),
	// и если оставить прежний снапшот рядом с чужим дневником, они вскроются тем
	// же ключом и применятся молча — база открылась бы состоянием, которого не
	// было ни в пакете, ни на устройстве. Остатки ротации и черновик снапшота
	// трогать не нужно: хранилище убирает их при открытии, а черновик не читается.
	// Обрыв между удалением и записью оставит пустой каталог — это восстановление
	// повторяемо (пакет цел), в отличие от молча перемешанного состояния.
	for _, name := range vaultBackupFiles {
		if err = os.Remove(filepath.Join(e.dataDir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rpc: прежний %s не убрать: %w", name, err)
		}
	}
	for _, f := range files {
		if err = os.WriteFile(filepath.Join(e.dataDir, f.name), f.data, 0600); err != nil {
			return fmt.Errorf("rpc: %s не восстановлен в %s: %w", f.name, e.dataDir, err)
		}
	}

	manifest, err := backupManifestProto(header, nil, provider.Name())
	if err != nil {
		return err
	}
	out.Manifest = manifest
	out.StatusMessage = fmt.Sprintf("восстановлено %d байт в %s из записи %s; список чанков в ответе пуст, потому что он лежит в зашифрованной части пакета — ключ ядра его не читает, а база уже открыта этим ключом; нужен StartNode",
		header.TotalPlain, e.dataDir, newest.Slot)
	action = backupActionRestored
	return nil
}
