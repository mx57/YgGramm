// File: core/pkg/fingerprint/measure.go
//
// Scan разбирает байты, которые в зеркало принёс кто-то другой. Для полевого
// вопроса «что про нас видит посредник» нужен противоположный конец: ровно те
// байты, которые отправляет ЭТОТ клиент, и ровно те границы, на которые они
// разложены. Recorder даёт оба.
//
// Что он честно измеряет, а что — нет:
//
//   - Снимается последовательность байтов, реально переданных в сокет (запись
//     учитывается только после удачного Write, и учитывается ровно то число
//     байт, которое сокет принял: частичная запись не притворяется полной).
//   - Границы между элементами Segments — это границы ВЫЗОВОВ Write, а не TCP-
//     сегменты на проводе. Ядро может склеить два маленьких вызова в один сегмент
//     (Nagle) или разбить один. Поэтому «hello ушёл двумя кусками» здесь
//     означает «мог уйти двумя кусками», и так же это должен читать любой, кто
//     судит про обход: проверка того, что посредник реально увидел два сегмента,
//     лежит вне этого файла (см. cmd/tspuprobe — там ответ посредника и есть
//     измерение).
//   - Разрез внутри одной TLS-записи (первые 5 байт отдельно от остального)
//     RecordVersions не увидит по построению: версии записей снимаются из
//     заголовков байтового потока, а поток от этого не меняется. Отсюда следует
//     проверка, которую держит measure_test.go: тот же байтовый поток, порезанный
//     по-другому, даёт ровно те же JA3, JA4 и разбор полей. Если это «улучшить»
//     до разных печаток, смысл потеряют и тест, и весь обход.
package fingerprint

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
)

// MaxHelloCapture — потолок снимка. ClientHello даже с постквантовыми ключами
// занимает единицы килобайт, а всё, что клиент пишет после рукопожатия, —
// зашифрованные записи, которые здесь снимаются только чтобы не раздувать буфер. Потолок нужен, чтобы
// соединение, где hello так и не собрался, не превращало снимок в бесконечный
// буфер.
const MaxHelloCapture = 64 << 10

// Recorder — обёртка над исходящим соединением, снимающая первый ClientHello и
// границы записей до него. Нулевое значение негодно, используй NewRecorder.
type Recorder struct {
	net.Conn

	mu       sync.Mutex
	capture  []byte
	segments []int
	hello    *ClientHello
	fail     error
	done     bool
}

// NewRecorder оборачивает соединение. Отдавать наружу нужно именно Recorder, а
// не исходный net.Conn: иначе фрагментатор, который лежит НАД ним, разрежет
// запись, а границы никто не запишет.
func NewRecorder(c net.Conn) *Recorder {
	return &Recorder{Conn: c}
}

func (r *Recorder) Write(b []byte) (int, error) {
	n, err := r.Conn.Write(b)
	if n > 0 {
		r.record(b[:n])
	}
	return n, err
}

func (r *Recorder) record(b []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done || r.fail != nil {
		return
	}
	if len(r.capture)+len(b) > MaxHelloCapture {
		r.fail = fmt.Errorf("fingerprint: снимок превысил потолок %d байт, целого ClientHello в нём нет", MaxHelloCapture)
		return
	}
	r.capture = append(r.capture, b...)
	r.segments = append(r.segments, len(b))
	if h, ok, err := Scan(r.capture); err != nil && !errors.Is(err, ErrShort) {
		r.fail = err
	} else if ok {
		r.hello = h
		r.done = true
	}
}

// Hello возвращает разобранный исходящий ClientHello. Ошибкой называется и
// битый поток, и случай, когда снятых байтов на целый hello не хватает: «соединение
// открылось, но отпечатка нет» — это разные утверждения, и сводить их к пустому
// значению нельзя.
func (r *Recorder) Hello() (*ClientHello, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hello != nil {
		return r.hello, nil
	}
	if r.fail != nil {
		return nil, r.fail
	}
	_, ok, err := Scan(r.capture)
	if err != nil && !errors.Is(err, ErrShort) {
		r.fail = err
		return nil, err
	}
	if ok {
		// Практически недостижимо: record() закрепляет hello на первом же собранном
		// кадре. Ветка держится потому, что «разобрано» не должно зависеть от того,
		// каким вызовом его спросили.
		h, _, _ := Scan(r.capture)
		r.hello = h
		return h, nil
	}
	return nil, fmt.Errorf("fingerprint: нет целого ClientHello в %d байт снимка", len(r.capture))
}

// Segments — длины записей, ушедших в сокет, от первой до той, которой hello
// довершился. Копия: вызывающий не должен иметь права двигать чужие границы.
func (r *Recorder) Segments() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]int, len(r.segments))
	copy(out, r.segments)
	return out
}

// Captured — снятые байты (копия). Нужен, чтобы пересчитать печатку вторым
// независимым кодом и чтобы найти смещение SNI для профиля разреза.
func (r *Recorder) Captured() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, len(r.capture))
	copy(out, r.capture)
	return out
}

// SNIByteOffset ищет первый байт имени хоста в снятых байтах hello и возвращает
// индекс, на котором это имя начинается. Смещение нужно профилю «разрез до SNI»:
// чтобы первый кусок не содержал ни одного байта имени, резать надо ровно по нему.
//
// Имя ищется как подстрока, и это предположение названо, а не молчит: в
// ClientHello hostname встречается ровно один раз (расширение server_name), так
// что совпадение байтов с key_share или с паддингом — это не норма, а поломка
// разбора. Поэтому функция требует, чтобы имя встречалось ровно один раз и чтобы
// перед ним стояла структура host_name из RFC 6066, и ошибкой отвергает и
// отсутствие, и повтор, и случайное совпадение.
func SNIByteOffset(stream []byte, name string) (int, error) {
	if name == "" {
		return 0, errors.New("fingerprint: пустое имя для поиска смещения SNI")
	}
	needle := []byte(name)
	at := bytes.Index(stream, needle)
	if at < 0 {
		return 0, fmt.Errorf("fingerprint: имя %q в снятых байтах не найдено", name)
	}
	if again := bytes.Index(stream[at+1:], needle); again >= 0 {
		return 0, fmt.Errorf("fingerprint: имя %q встречается в снимке дважды (на %d и %d), разрез по нему неоднозначен",
			name, at, at+1+again)
	}
	// Перед именем по RFC 6066 лежит структура server_name: байт типа host_name
	// (0x00) и двухбайтная длина самого имени. Проверка именно её, а не «любого
	// байта перед хостом»: имя предшествует 0x0c, а не 0x00, и путать это — значит
	// резать hello не там, где SNI.
	if at < 3 || stream[at-3] != 0x00 || int(uint16(stream[at-2])<<8|uint16(stream[at-1])) != len(needle) {
		return 0, fmt.Errorf("fingerprint: перед %q нет структуры host_name (0x00 + длина) — нашлось не SNI", name)
	}
	return at, nil
}

// Fragmented — признаком служит длина RecordVersions: hello пришёл более чем в
// одной TLS-записи. Отдельно от печатки: именно это поле отличает «несколько
// записей» от «одна запись, порезанная по TCP».
func (h *ClientHello) Fragmented() bool { return len(h.RecordVersions) > 1 }
