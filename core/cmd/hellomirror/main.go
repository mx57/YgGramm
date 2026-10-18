// File: core/cmd/hellomirror/main.go
//
// hellomirror — TCP-зеркало для замера отпечатка чужого TLS-клиента. Нужен ровно
// для одного: снять реальные байты ClientHello браузера (или любого другого
// клиента) и посчитать по ним JA3 и JA4 тем же кодом, которым ядро считает
// отпечаток собственного tls://-транспорта. Без этого сравнение «мы против
// браузера» осталось бы числами из чужих статей.
//
//	go run ./cmd/hellomirror -up example.com:443
//
// Затем в браузере открыть https://localhost:8443/ (имя, а не адрес — иначе
// клиент не положит SNI, и ветка «d» в JA4-приставке не проверится). Сервер
// возврата не нужен: сертификат для «localhost» почти наверняка невалиден, а hello
// уходит первым полётом, до любой проверки сертификата. С -up зеркало пересылает
// байты настоящему серверу и соединение живёт как обычное; без -up оно закрыто
// сразу после замера.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"ygggram/core/pkg/fingerprint"
)

func main() {
	listen := flag.String("listen", ":8443", "адрес, куда клиент должен подключаться вместо реального сервера (пустой хост = и IPv4, и IPv6)")
	upstream := flag.String("up", "", "адрес настоящего TLS-сервера для пересылки, например example.com:443; пусто — только снять замер")
	maxHello := flag.Int("max", 16384, "сколько байт потока максимум разобрать, если client_hello так и не собрался")
	flag.Parse()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	if *upstream == "" {
		log.Printf("слушаю %s: режим замера, соединение закрывается после client_hello", ln.Addr())
	} else {
		log.Printf("слушаю %s, пересылаю на %s", ln.Addr(), *upstream)
	}

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(client, *upstream, *maxHello)
	}
}

func handle(client net.Conn, upstream string, maxHello int) {
	defer client.Close()

	var up net.Conn
	if upstream != "" {
		dialed, err := net.Dial("tcp", upstream)
		if err != nil {
			log.Printf("%s: uplink %s: %v", client.RemoteAddr(), upstream, err)
			return
		}
		up = dialed
		defer up.Close()
		// Ответ летит отдельным циклом и не блокируется разбором: зеркало ничего
		// не решает на своём пути.
		go func() { _, _ = io.Copy(client, up) }()
	}

	buf := make([]byte, 4096)
	var seen []byte
	helloDone := false
	for len(seen) < maxHello {
		n, err := client.Read(buf)
		if n > 0 {
			if up != nil {
				if _, werr := up.Write(buf[:n]); werr != nil {
					return
				}
			}
			if !helloDone {
				seen = append(seen, buf[:n]...)
				h, ok, perr := fingerprint.Scan(seen)
				switch {
				case ok:
					printHello(client.RemoteAddr(), h)
					helloDone = true
				case perr != nil && !errors.Is(perr, fingerprint.ErrShort):
					log.Printf("%s: это не client_hello: %v", client.RemoteAddr(), perr)
					return
				}
				// Без апстрима дальше читать нечего: разбор закончен, соединение
				// кому-то другой не нужен. Повторно Scan не вызывается — иначе
				// следующая за запись (application data) выглядела бы как «это не
				// client_hello».
				if up == nil {
					return
				}
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("%s: read: %v", client.RemoteAddr(), err)
			}
			return
		}
	}
	if !helloDone {
		log.Printf("%s: за %d байт client_hello не собрался", client.RemoteAddr(), len(seen))
	}
}

func printHello(addr net.Addr, h *fingerprint.ClientHello) {
	fmt.Println("---- client_hello от", addr, "----")
	fmt.Println(h.Summary())
	fmt.Println("JA3 spec:", h.JA3Spec())
	fmt.Println("JA3 hash:", h.JA3())
	fmt.Println("JA4:     ", h.JA4())
	fmt.Println("JA4_r:   ", h.JA4Raw())
	fmt.Println()
}
