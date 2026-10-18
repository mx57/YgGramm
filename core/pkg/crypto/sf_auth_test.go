// File: core/pkg/crypto/sf_auth_test.go
package crypto

import (
	"crypto/ed25519"
	"testing"
	"time"
)

// pullOf — годный запрос к релею от этой идентичности: ящик, эпоха и подпись
// согласованы между собой.
func pullOf(t *testing.T, master *MasterIdentity, epochDay, timestampS int64) ([32]byte, []byte) {
	t.Helper()
	mailbox := ComputeMailboxID(master.MasterX25519.PublicKey().Bytes(), epochDay)
	sig, err := BuildSFPullSignature(master, mailbox, epochDay, timestampS)
	if err != nil {
		t.Fatalf("BuildSFPullSignature: %v", err)
	}
	return mailbox, sig
}

// TestSFPullSignatureRoundTrip — подпись годного запроса проходит проверку, а
// домен в подписываемых байтах делает её непереиспользуемой.
func TestSFPullSignatureRoundTrip(t *testing.T) {
	master := identityOf(t, "crush")
	epoch := time.Now().Unix() / 86400
	now := time.Now().Unix()
	mailbox, sig := pullOf(t, master, epoch, now)

	if err := VerifySFPullSignature(master.MasterEd25519.Public().(ed25519.PublicKey),
		mailbox, epoch, now, sig, now); err != nil {
		t.Fatalf("годная подпись отвергнута: %v", err)
	}
	// Тот же ящик, выведенный из этого же ключа, обязан совпасть с recomputed:
	// иначе релей не сможет проверить принадлежность почты.
	if err := MailboxBelongsTo(master.MasterX25519.PublicKey().Bytes(), epoch, mailbox); err != nil {
		t.Fatalf("ящик не признал своего владельца: %v", err)
	}
}

// TestSFPullSignatureRejects — каждый способ подделать запрос обязана ловить
// конкретная проверка, а не «как-то не совпало».
func TestSFPullSignatureRejects(t *testing.T) {
	master := identityOf(t, "crush")
	other := identityOf(t, "hint")
	epoch := time.Now().Unix() / 86400
	now := time.Now().Unix()
	mailbox, sig := pullOf(t, master, epoch, now)
	pub := master.MasterEd25519.Public().(ed25519.PublicKey)
	otherPub := other.MasterEd25519.Public().(ed25519.PublicKey)

	if err := VerifySFPullSignature(otherPub, mailbox, epoch, now, sig, now); err == nil {
		t.Fatal("подпись чужим мастер-ключом прошла")
	}
	if err := VerifySFPullSignature(pub, mailbox, epoch+1, now, sig, now); err == nil {
		t.Fatal("подпись не проходит, если эпоха в запросе изменена")
	}
	tampered := mailbox
	tampered[0] ^= 1
	if err := VerifySFPullSignature(pub, tampered, epoch, now, sig, now); err == nil {
		t.Fatal("подпись не проходит, если ящик в запросе изменён")
	}
	if err := VerifySFPullSignature(pub, mailbox, epoch, now+2*SFPullWindowSeconds, sig, now); err == nil {
		t.Fatal("просроченная метка времени принята")
	}
	// Свежая метка из будущего тоже не годится: окно симметрично, иначе
	// запрос с заранее выставленным часом жил бы вдвое дольше разрешённого.
	if err := VerifySFPullSignature(pub, mailbox, epoch, now+SFPullWindowSeconds+10, sig, now); err == nil {
		t.Fatal("метка времени из будущего принята")
	}
	if err := VerifySFPullSignature(pub, mailbox, epoch, now, sig[:ed25519.SignatureSize-1], now); err == nil {
		t.Fatal("обрезанная подпись принята")
	}
	if err := VerifySFPullSignature(pub[:31], mailbox, epoch, now, sig, now); err == nil {
		t.Fatal("ed25519-ключ неверной длины принят")
	}
}

// TestMailboxBelongsTo — ящик обязан пересчитываться из ключа, а не
// приниматься на веру: иначе достаточно знать открытый ключ человека, чтобы
// забрать его почту с релея.
func TestMailboxBelongsTo(t *testing.T) {
	master := identityOf(t, "crush")
	other := identityOf(t, "hint")
	epoch := time.Now().Unix() / 86400
	mailbox := ComputeMailboxID(master.MasterX25519.PublicKey().Bytes(), epoch)

	if err := MailboxBelongsTo(master.MasterX25519.PublicKey().Bytes(), epoch, mailbox); err != nil {
		t.Fatalf("свой ящик отвергнут: %v", err)
	}
	if err := MailboxBelongsTo(other.MasterX25519.PublicKey().Bytes(), epoch, mailbox); err == nil {
		t.Fatal("чужой мастер-X25519 признан владельцем ящика")
	}
	if err := MailboxBelongsTo(master.MasterX25519.PublicKey().Bytes(), epoch-1, mailbox); err == nil {
		t.Fatal("ящик принят в другой эпохе")
	}
	if err := MailboxBelongsTo(master.MasterX25519.PublicKey().Bytes()[:31], epoch, mailbox); err == nil {
		t.Fatal("укороченный ключ принят")
	}
	// Нуль-дополненный «ящик» не должен совпасть ни с чем осмысленным.
	var zero [32]byte
	if err := MailboxBelongsTo(master.MasterX25519.PublicKey().Bytes(), epoch, zero); err == nil {
		t.Fatal("нулевой идентификатор признан ящиком")
	}
}
