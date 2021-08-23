package main

import (
	"bytes"
	"io"
	"log"
	"regexp"
	"sync"

	"github.com/gofiber/fiber/v2/utils"
)

const URLRegexp = `(?i)(\/\/([a-z0-9]+(-*[a-z0-9]+)*\.)+(arpa|root|aero|biz|cat|com|coop|edu|gov|info|int|jobs|mil|mobi|museum|name|net|org|pro|tel|travel|ac|ad|ae|af|ag|ai|al|am|an|ao|aq|ar|as|at|au|aw|ax|az|ba|bb|bd|be|bf|bg|bh|bi|bj|bm|bn|bo|br|bs|bt|bv|bw|by|bz|ca|cc|cd|cf|cg|ch|ci|ck|cl|cm|cn|co|cr|cu|cv|cx|cy|cz|dev|de|dj|dk|dm|do|dz|ec|ee|eg|er|es|et|eu|fi|fj|fk|fm|fo|fr|ga|gb|gd|ge|gf|gg|gh|gi|gl|gm|gn|gp|gq|gr|gs|gt|gu|gw|gy|hk|hm|hn|hr|ht|hu|id|ie|il|im|in|io|iq|ir|is|it|je|jm|jo|jp|ke|kg|kh|ki|km|kn|kr|kw|ky|kz|la|lb|lc|li|lk|lr|ls|lt|lu|lv|ly|ma|mc|md|mg|mh|mk|ml|mm|mn|mo|mp|mq|mr|ms|mt|mu|mv|mw|mx|my|mz|na|nc|ne|nf|ng|ni|nl|no|np|nr|nu|nz|om|pa|pe|pf|pg|ph|pk|pl|pm|pn|pr|ps|pt|pw|py|qa|re|ro|ru|rw|sa|sb|sc|sd|se|sg|sh|si|sj|sk|sl|sm|sn|so|sr|st|su|sv|sy|sz|tc|td|tf|tg|th|tj|tk|tl|tm|tn|to|tp|tr|tt|tv|tw|tz|ua|ug|uk|um|us|uy|uz|va|vc|ve|vg|vi|vn|vu|wf|ws|ye|yt|yu|za|zm|zw))`
const PartialURLRegexp = `(?i)(\/(\/([a-z0-9]+(-[a-z0-9])*)?)?$)`

var (
	urlRegexp        = regexp.MustCompile(URLRegexp)
	partialURLRegexp = regexp.MustCompile(PartialURLRegexp)
	buffPool         = newBufferPool()
)

type BufferPool struct {
	pool sync.Pool
}

func newBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *BufferPool) Put(buff *bytes.Buffer) {
	buff.Reset()
	p.pool.Put(buff)
}

type RegexProcessor interface {
	// Process reads next n bytes from r into the buff, processes it and writes into w.
	// Returns io.EOF when no more input is available. It discards bytes from the buffer
	// that were written to w.
	Process(w io.Writer, r io.Reader, n int, buff *bytes.Buffer) error
	// ProcessAll reads all bytes from r, processes it and writes into w.
	// A successful ProcessAll returns err == nil, not err == io.EOF.
	// Because ProcessAll is defined to read from src until EOF,
	// it does not treat an EOF from Read as an error to be reported.
	ProcessAll(w io.Writer, r io.Reader) error
}

func newURLRegexProcessor(convertDomain convertDomainFunc) RegexProcessor {
	return &regexProcessor{
		re:                  urlRegexp,
		partialSuffixRegexp: partialURLRegexp,
		convertDomain:       convertDomain,
	}
}

// TODO remove integrity attribute
func newHTMLRegexProcessor(conv DomainConverter, jsHookScript string) RegexProcessor {
	htmlRegexp := regexp.MustCompile(URLRegexp + `|(crossorigin="anonymous")|(rel="manifest")|(<head>)`)
	return &regexProcessor{
		re:                  htmlRegexp,
		partialSuffixRegexp: partialURLRegexp,
		convertDomain: func(domain string) string {
			return conv.ToProxyDomain(domain)
		},
		replaceMap: map[string]string{
			`crossorigin="anonymous"`: "",
			`rel="manifest"`:          `rel="manifest" crossorigin="use-credentials"`,
			`<head>`:                  `<head><script>` + jsHookScript + `</script>`,
		},
	}
}

type convertDomainFunc func(domain string) string

type regexProcessor struct {
	re                  *regexp.Regexp
	partialSuffixRegexp *regexp.Regexp
	replaceMap          map[string]string
	convertDomain       convertDomainFunc
}

//nolint:errcheck
func (p *regexProcessor) Process(w io.Writer, r io.Reader, count int, buff *bytes.Buffer) error {
	n, err := buff.ReadFrom(io.LimitReader(r, int64(count)))
	if n == 0 {
		err = io.EOF
	}
	bytearr := buff.Bytes()
	foundIndex := p.re.FindAllIndex(bytearr, -1)
	start := 0
	for _, pair := range foundIndex {
		w.Write(bytearr[start:pair[0]])
		// handle case: //bbc.ae -> ro
		if pair[1] == count {
			start = pair[0]
			break
		}

		found := string(bytearr[pair[0]:pair[1]])
		if replaced, ok := p.replaceMap[found]; ok {
			w.Write(utils.UnsafeBytes(replaced))
		} else {
			w.Write([]byte("//"))
			w.Write([]byte(p.convertDomain(found[2:])))
		}
		start = pair[1]
	}
	// advance the buffer by the number of processed bytes
	if start > 0 {
		advanceBuffer(buff, start)
	} else {
		pair := p.partialSuffixRegexp.FindIndex(bytearr)
		if pair != nil {
			w.Write(buff.Bytes()[:pair[0]])
			advanceBuffer(buff, pair[0])
		} else {
			w.Write(buff.Bytes())
			buff.Reset()
		}
	}
	return err
}

// advanceBuffer advances the buff buffer by the num bytes
func advanceBuffer(buff *bytes.Buffer, num int) {
	buff.Next(num)
	// move buffer from num offset to 0
	bytearr := buff.Bytes()
	buff.Reset()
	buff.Write(bytearr)
}

//nolint:errcheck
func (p *regexProcessor) ProcessAll(w io.Writer, r io.Reader) (err error) {
	buff := buffPool.Get()
	defer buffPool.Put(buff)
	const bufSize = 4096
	for {
		if err = p.Process(w, r, bufSize, buff); err != nil {
			w.Write(buff.Bytes())
			if err != io.EOF {
				log.Println("io error", err)
			} else {
				err = nil
			}
			return
		}
	}
}

type replaceRegexReader struct {
	delegate io.Reader
	proc     RegexProcessor
	buff     *bytes.Buffer
	output   *bytes.Buffer
	mu       sync.RWMutex
	closed   bool
}

func NewReplaceRegexReader(r io.Reader, proc RegexProcessor) io.ReadCloser {
	return &replaceRegexReader{
		delegate: r, proc: proc,
		buff: buffPool.Get(), output: buffPool.Get(),
	}
}

//nolint:errcheck
func (r *replaceRegexReader) Read(p []byte) (n int, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	const bufSize = 4096
	for {
		count := copy(p[n:], r.output.Bytes())
		n += count
		advanceBuffer(r.output, count)
		if n == len(p) || err != nil {
			return
		}
		if err = r.proc.Process(r.output, r.delegate, bufSize, r.buff); err != nil {
			r.output.Write(r.buff.Bytes())
		}
	}
}

func (r *replaceRegexReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	buffPool.Put(r.buff)
	buffPool.Put(r.output)
	return nil
}
