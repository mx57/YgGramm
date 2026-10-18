// File: core/pkg/crypto/sf_auth.go
package crypto

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
)

// sfPullDomain — контекст подписи на запросе к S&F-релею. Он стоит в
// подписываемых байтах не для красоты: подпись без домена переехала бы из
// «забери мою почту» в «положи мою почту» или в любой другой протокол, где
// тот же мастер-Ed25519 что-то подписывает.
const sfPullDomain = "ygggram-sf-pull-v1"

// SFPullWindowSeconds — насколько метка времени в запросе обязана быть свежей.
// Дальше подпись перестаёт считаться годной: её нельзя снять с чужого
// перехваченного пакета и использовать позже.
const SFPullWindowSeconds = 300

// SFPullAuthPayload — байты, которые подписывает получатель, забирая почту:
// домен || ящик || эпоха-день || метка времени.
func SFPullAuthPayload(mailboxID [32]byte, epochDay, timestampS int64) []byte {
	out := make([]byte, 0, len(sfPullDomain)+32+8+8)
	out = append(out, sfPullDomain...)
	out = append(out, mailboxID[:]...)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(epochDay))
	out = append(out, b[:]...)
	binary.BigEndian.PutUint64(b[:], uint64(timestampS))
	out = append(out, b[:]...)
	return out
}

// BuildSFPullSignature подписывает запрос мастер-Ed25519 идентичности, которой
// принадлежит ящик.
func BuildSFPullSignature(master *MasterIdentity, mailboxID [32]byte, epochDay, timestampS int64) ([]byte, error) {
	if master == nil {
		return nil, errors.New("sealed: подпись запроса к релею требует мастер-идентичность")
	}
	return ed25519.Sign(master.MasterEd25519, SFPullAuthPayload(mailboxID, epochDay, timestampS)), nil
}

// VerifySFPullSignature проверяет подпись и её свежесть. Сама по себе она
// ничего не говорит о том, что подписант владеет ящиком, — это обязана
// доказать связка из IdentityProof: вызывающий обязан сверить, что ed25519
// здесь тот же, что подписал masterX25519, из которого вычислен mailboxID.
func VerifySFPullSignature(ed25519Pub ed25519.PublicKey, mailboxID [32]byte, epochDay, timestampS int64, sig []byte, nowS int64) error {
	if len(ed25519Pub) != ed25519.PublicKeySize {
		return errors.New("sealed: в запросе к релею ed25519-ключ неверной длины")
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("sealed: подпись запроса к релею %d байт, ожидалось %d", len(sig), ed25519.SignatureSize)
	}
	delta := nowS - timestampS
	if delta > SFPullWindowSeconds || delta < -SFPullWindowSeconds {
		return fmt.Errorf("sealed: метка времени запроса к релею отстаёт на %d с, окно %d с", delta, SFPullWindowSeconds)
	}
	if !ed25519.Verify(ed25519Pub, SFPullAuthPayload(mailboxID, epochDay, timestampS), sig) {
		return errors.New("sealed: подпись запроса к релею не проходит")
	}
	return nil
}

// MailboxBelongsTo verifies, что ящик выведен именно из этого мастер-X25519 в
// этой эпохе. Релей не хранит соответствие «ящик -> ключ»: он пересчитывает
// индекс и сравнивает, поэтому подделанная привязка не проходит.
func MailboxBelongsTo(masterX25519 []byte, epochDay int64, mailboxID [32]byte) error {
	if len(masterX25519) != 32 {
		return fmt.Errorf("sealed: мастер-X25519 на %d байт, нужно 32", len(masterX25519))
	}
	if got := ComputeMailboxID(masterX25519, epochDay); got != mailboxID {
		return errors.New("sealed: ящик не выведен из заявленного мастер-X25519 в этой эпохе")
	}
	return nil
}
