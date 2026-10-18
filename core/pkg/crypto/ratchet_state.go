// File: core/pkg/crypto/ratchet_state.go
package crypto

import (
	"crypto/ecdh"
	"fmt"
	"time"
)

// Границы кэша пропущенных ключей. Живой сеанс и восстановленный из базы
// обязаны пользоваться одними числами, иначе перезапуск менял бы поведение
// рэтчета.
const (
	maxSkippedKeys = 2000
	skippedKeyTTL  = 7 * 24 * time.Hour
)

// SkippedKeyState — ключевое сообщение, которое рэтчет вывел заранее из-за
// разрыва в приёмной цепочке и отложил до припозднившегося пакета. ID — ровно
// та строка `hex(DHR)-номер`, под которой ключ лежит в живой карте, поэтому
// импорт возвращает её без пересчёта.
type SkippedKeyState struct {
	ID          string `json:"id"`
	MessageKey  []byte `json:"mk"`
	CreatedAtMS int64  `json:"t"`
}

// SessionState — всё изменяемое состояние одного сеанса Double Ratchet: его
// хватит, чтобы продолжить переписку после рестарта ядра, и ничего лишнего
// (мастер-ключи узла сюда не попадают).
//
// Запись этого на диск означает, что стойкость к компрометации прошлых
// сообщений держится теперь на ключе хранилища: у кого файл базы и passphrase,
// тот раскрывает и будущие цепочки. Так устроен любой мессенджер, где
// переписка переживает перезапуск (у Signal состояние сессий живёт в его БД), —
// альтернатива один раз потерять все открытые диалоги.
type SessionState struct {
	// DHSPrivateKey — приватная половина текущего локального DH-ключа. Её
	// публичная часть выводима, поэтому в состоянии не дублируется.
	DHSPrivateKey []byte `json:"dhs_priv"`
	// DHRPublicKey — текущий удалённый DH-ключ. Пустой до первого принятого
	// пакета: ровно тогда сеанс ещё не сделал ни одного DH-шага в приём.
	DHRPublicKey []byte `json:"dhr,omitempty"`

	RootKey      []byte `json:"rk"`
	SendChainKey []byte `json:"ck_send,omitempty"`
	RecvChainKey []byte `json:"ck_recv,omitempty"`

	SendSequence    uint32 `json:"ns"`
	RecvSequence    uint32 `json:"nr"`
	PrevChainLength uint32 `json:"pcl"`

	Skipped []SkippedKeyState `json:"sk,omitempty"`
}

// ExportState снимает состояние сеанса для записи в хранилище. Снимок берётся
// под тем же мьютексом, что шифрование и расшифрование, поэтому записать можно
// только согласованный шаг цепочки, а не половину сдвига.
func (s *DoubleRatchetSession) ExportState() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := SessionState{
		DHSPrivateKey:   s.dhsPrivateKey.Bytes(),
		RootKey:         cloneKey(s.rootKey),
		SendChainKey:    cloneKey(s.sendingChainKey),
		RecvChainKey:    cloneKey(s.recvChainKey),
		SendSequence:    s.sendSequence,
		RecvSequence:    s.recvSequence,
		PrevChainLength: s.prevChainLength,
	}
	if s.dhrPublicKey != nil {
		st.DHRPublicKey = s.dhrPublicKey.Bytes()
	}
	if len(s.skippedKeys) > 0 {
		st.Skipped = make([]SkippedKeyState, 0, len(s.skippedKeys))
		for id, item := range s.skippedKeys {
			key := make([]byte, len(item.MessageKey))
			copy(key, item.MessageKey[:])
			st.Skipped = append(st.Skipped, SkippedKeyState{
				ID:          id,
				MessageKey:  key,
				CreatedAtMS: item.CreatedAt.UnixMilli(),
			})
		}
	}
	return st
}

// SessionFromState восстанавливает сеанс из сохранённого состояния. Проверка
// жёсткая: состояние приходит из файла, который мог быть испорчен или подправлен,
// а молча принятый мусор превратился бы в сеанс, который никогда ничего не
// раскроет.
func SessionFromState(st SessionState) (*DoubleRatchetSession, error) {
	dhsPriv, err := ecdh.X25519().NewPrivateKey(st.DHSPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("ratchet: восстановленный DH-приватный ключ: %w", err)
	}

	rootKey, err := fixedKey(st.RootKey, "ключ корня")
	if err != nil {
		return nil, err
	}

	session := &DoubleRatchetSession{
		dhsPrivateKey:   dhsPriv,
		dhsPublicKey:    dhsPriv.PublicKey(),
		rootKey:         rootKey,
		sendSequence:    st.SendSequence,
		recvSequence:    st.RecvSequence,
		prevChainLength: st.PrevChainLength,
		skippedKeys:     make(map[string]SkippedKey, len(st.Skipped)),
		maxSkipped:      maxSkippedKeys,
		keyTTL:          skippedKeyTTL,
	}

	if len(st.DHRPublicKey) > 0 {
		dhr, err := ecdh.X25519().NewPublicKey(st.DHRPublicKey)
		if err != nil {
			return nil, fmt.Errorf("ratchet: восстановленный удалённый DH-ключ: %w", err)
		}
		session.dhrPublicKey = dhr
	}
	// Цепочка отправителя пустая — это законный сеанс получателя до первого
	// ответа: Encrypt сам откажется шифровать, пока DH-шаг не завёл цепочку.
	if len(st.SendChainKey) > 0 {
		ck, err := fixedKey(st.SendChainKey, "ключ отправной цепочки")
		if err != nil {
			return nil, err
		}
		session.sendingChainKey = ck
	}
	if len(st.RecvChainKey) > 0 {
		ck, err := fixedKey(st.RecvChainKey, "ключ приёмной цепочки")
		if err != nil {
			return nil, err
		}
		session.recvChainKey = ck
	}

	if len(st.Skipped) > maxSkippedKeys {
		return nil, fmt.Errorf("ratchet: в состоянии %d пропущенных ключей, предел %d", len(st.Skipped), maxSkippedKeys)
	}
	for _, item := range st.Skipped {
		if item.ID == "" {
			return nil, fmt.Errorf("ratchet: пропущенный ключ без идентификатора")
		}
		mk, err := fixedKey(item.MessageKey, "ключ пропущенного сообщения")
		if err != nil {
			return nil, err
		}
		created := time.UnixMilli(item.CreatedAtMS)
		if item.CreatedAtMS <= 0 {
			created = time.Now()
		}
		var key [32]byte
		copy(key[:], mk)
		session.skippedKeys[item.ID] = SkippedKey{MessageKey: key, CreatedAt: created}
	}

	return session, nil
}

func cloneKey(in []byte) []byte {
	if in == nil {
		return nil
	}
	return append([]byte(nil), in...)
}

func fixedKey(in []byte, name string) ([]byte, error) {
	if len(in) != 32 {
		return nil, fmt.Errorf("ratchet: %s в состоянии должен быть 32 байта, а не %d", name, len(in))
	}
	return append([]byte(nil), in...), nil
}
