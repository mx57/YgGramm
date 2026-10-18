// File: core/pkg/crypto/node_binding.go
package crypto

import (
	"crypto/ed25519"
	"errors"
	"fmt"
)

// nodeKeyBindingDomain — домен сертификата ключа ноды. Подпись без домена было
// бы можно перенести из одного протокола в другой, поэтому префикс входит в
// подписываемые байты на обеих сторонах.
const nodeKeyBindingDomain = "ygggram-node-key-bind-v1"

// nodeKeyBindingMessage — единственное, что подписывает мастер-Ed25519 в
// сертификате ноды: мастер-X25519 и публичный ключ узла Yggdrasil.
//
// Порядок важен: оба ключа фиксируются по длине, поэтому разбор однозначен, а
// произвольная пара байтов не соберёт тот же самый текст.
func nodeKeyBindingMessage(masterX25519, nodeEd25519 []byte) ([]byte, error) {
	if len(masterX25519) != 32 {
		return nil, fmt.Errorf("node binding: мастер-X25519 %d байт, нужно 32", len(masterX25519))
	}
	if len(nodeEd25519) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("node binding: ключ ноды %d байт, нужно %d", len(nodeEd25519), ed25519.PublicKeySize)
	}
	msg := make([]byte, 0, len(nodeKeyBindingDomain)+32+ed25519.PublicKeySize)
	msg = append(msg, nodeKeyBindingDomain...)
	msg = append(msg, masterX25519...)
	msg = append(msg, nodeEd25519...)
	return msg, nil
}

// BuildNodeKeyProof заверяет ключ ноды от имени идентичности. Сам ключ узла
// ничего не знает о владельце, поэтому подписывает пару «мастер-X25519 ↔ ключ
// узла» именно мастер-ключ: получатель видит, что адрес доставки и адрес X3DH
// выбрал один и тот же владелец.
func BuildNodeKeyProof(master *MasterIdentity, nodeEd25519 ed25519.PublicKey) ([]byte, error) {
	if master == nil {
		return nil, errors.New("node binding: нет мастер-идентичности")
	}
	msg, err := nodeKeyBindingMessage(master.MasterX25519.PublicKey().Bytes(), nodeEd25519)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(master.MasterEd25519, msg), nil
}

// VerifyNodeKeyProof проверяет сертификат под мастер-Ed25519 из того же бандла,
// что подписывает signed prekey. Отказ означает, что ключ ноды этого контакта
// не принадлежит заявленной идентичности — писать такому адресу нельзя.
func VerifyNodeKeyProof(masterEd25519 ed25519.PublicKey, masterX25519, nodeEd25519, proof []byte) error {
	if len(masterEd25519) != ed25519.PublicKeySize {
		return fmt.Errorf("node binding: мастер-Ed25519 %d байт, нужно %d", len(masterEd25519), ed25519.PublicKeySize)
	}
	if len(proof) != ed25519.SignatureSize {
		return fmt.Errorf("node binding: сертификат %d байт, нужно %d", len(proof), ed25519.SignatureSize)
	}
	msg, err := nodeKeyBindingMessage(masterX25519, nodeEd25519)
	if err != nil {
		return err
	}
	if !ed25519.Verify(masterEd25519, msg, proof) {
		return errors.New("node binding: сертификат не подписан этим мастер-Ed25519 поверх этой пары ключей")
	}
	return nil
}
