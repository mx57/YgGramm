// File: core/pkg/rpc/frame.go
package rpc

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/crypto"

	yggproto "ygggram/core/proto"
)

// Кадр поверх оверлея. До Шага 6 на проводе лежал голый SealedEnvelope, и
// любой другой пакет означал для приёмного цикла «испорченный конверт».
// Служебному трафику S&F-релея нужна своя оболочка, и это один protobuf-oneof,
// а не самописный префикс-байт: различие видов читается тем же парсером, а
// неизвестный вариант отсеивается на месте, а не превращается в мусор.
//
// Байты кадра — то и только то, что ложится в сокет и в исходящую очередь:
// переотправка обязана повторять ровно эти байты, поэтому упаковку делает
// один помощник, а не два пути.

func chatFrame(env *yggproto.SealedEnvelope) ([]byte, error) {
	if env == nil {
		return nil, errors.New("rpc: пустой конверт в кадр не положить")
	}
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_Chat{Chat: env}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка chat-кадра: %w", err)
	}
	return out, nil
}

func sfPutFrame(req *yggproto.SfPut) ([]byte, error) {
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_SfPut{SfPut: req}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка sf_put-кадра: %w", err)
	}
	return out, nil
}

func sfPullFrame(req *yggproto.SfPull) ([]byte, error) {
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_SfPull{SfPull: req}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка sf_pull-кадра: %w", err)
	}
	return out, nil
}

func sfReplyFrame(reply *yggproto.SfReply) ([]byte, error) {
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_SfReply{SfReply: reply}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка sf_reply-кадра: %w", err)
	}
	return out, nil
}

func bundleReqFrame(req *yggproto.BundleReq) ([]byte, error) {
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_BundleReq{BundleReq: req}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка bundle_req-кадра: %w", err)
	}
	return out, nil
}

func bundleRespFrame(resp *yggproto.BundleResp) ([]byte, error) {
	out, err := proto.Marshal(&yggproto.WireFrame{Body: &yggproto.WireFrame_BundleResp{BundleResp: resp}})
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка bundle_resp-кадра: %w", err)
	}
	return out, nil
}

// deframe разбирает кадр и сообщает, что в нём лежит. Незнакомый вариант
// (пустой oneof) — ошибка, а не тишина: молча пройти мимо чужого пакета
// означает потерять его без следа в счётчике.
func deframe(payload []byte) (*yggproto.WireFrame, error) {
	frame := &yggproto.WireFrame{}
	if err := proto.Unmarshal(payload, frame); err != nil {
		return nil, fmt.Errorf("rpc: кадр не разбирается: %w", err)
	}
	if frame.GetBody() == nil {
		return nil, errors.New("rpc: пустой кадр оверлея")
	}
	return frame, nil
}

// proofToWire переносит связку мастер-ключей в открытый вид: релею она нужна
// именно в явном виде, чтобы привязать ящик к идентичности до того, как он
// отдаст чужую почту.
func proofToWire(p *crypto.IdentityProof) *yggproto.WireIdentityProof {
	if p == nil {
		return nil
	}
	return &yggproto.WireIdentityProof{
		MasterEd25519:   append([]byte(nil), p.MasterEd25519...),
		MasterX25519:    append([]byte(nil), p.MasterX25519...),
		Binding:         append([]byte(nil), p.Binding...),
		DeviceId:        p.DeviceID,
		DeviceName:      p.DeviceName,
		DeviceEd25519:   append([]byte(nil), p.DeviceEd25519...),
		DeviceX25519:    append([]byte(nil), p.DeviceX25519...),
		DeviceCreatedAt: p.DeviceCreatedAt,
		DeviceCertSig:   append([]byte(nil), p.DeviceCertSig...),
	}
}

func proofFromWire(p *yggproto.WireIdentityProof) *crypto.IdentityProof {
	if p == nil {
		return nil
	}
	return &crypto.IdentityProof{
		MasterEd25519:   p.GetMasterEd25519(),
		MasterX25519:    p.GetMasterX25519(),
		Binding:         p.GetBinding(),
		DeviceID:        p.GetDeviceId(),
		DeviceName:      p.GetDeviceName(),
		DeviceEd25519:   p.GetDeviceEd25519(),
		DeviceX25519:    p.GetDeviceX25519(),
		DeviceCreatedAt: p.GetDeviceCreatedAt(),
		DeviceCertSig:   p.GetDeviceCertSig(),
	}
}
