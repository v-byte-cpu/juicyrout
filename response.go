package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

const URLRegexp = `(\/\/([A-Za-z0-9]+(-[a-z0-9]+)*\.)+(arpa|root|aero|biz|cat|com|coop|edu|gov|info|int|jobs|mil|mobi|museum|name|net|org|pro|tel|travel|ac|ad|ae|af|ag|ai|al|am|an|ao|aq|ar|as|at|au|aw|ax|az|ba|bb|bd|be|bf|bg|bh|bi|bj|bm|bn|bo|br|bs|bt|bv|bw|by|bz|ca|cc|cd|cf|cg|ch|ci|ck|cl|cm|cn|co|cr|cu|cv|cx|cy|cz|dev|de|dj|dk|dm|do|dz|ec|ee|eg|er|es|et|eu|fi|fj|fk|fm|fo|fr|ga|gb|gd|ge|gf|gg|gh|gi|gl|gm|gn|gp|gq|gr|gs|gt|gu|gw|gy|hk|hm|hn|hr|ht|hu|id|ie|il|im|in|io|iq|ir|is|it|je|jm|jo|jp|ke|kg|kh|ki|km|kn|kr|kw|ky|kz|la|lb|lc|li|lk|lr|ls|lt|lu|lv|ly|ma|mc|md|mg|mh|mk|ml|mm|mn|mo|mp|mq|mr|ms|mt|mu|mv|mw|mx|my|mz|na|nc|ne|nf|ng|ni|nl|no|np|nr|nu|nz|om|pa|pe|pf|pg|ph|pk|pl|pm|pn|pr|ps|pt|pw|py|qa|re|ro|ru|rw|sa|sb|sc|sd|se|sg|sh|si|sj|sk|sl|sm|sn|so|sr|st|su|sv|sy|sz|tc|td|tf|tg|th|tj|tk|tl|tm|tn|to|tp|tr|tt|tv|tw|tz|ua|ug|uk|um|us|uy|uz|va|vc|ve|vg|vi|vn|vu|wf|ws|ye|yt|yu|za|zm|zw))`

type ResponseProcessor interface {
	Process(c *fiber.Ctx, resp *http.Response)
}

func NewResponseProcessor(conv DomainConverter) ResponseProcessor {
	re := regexp.MustCompile(URLRegexp)
	return &responseProcessor{conv: conv, urlRegexp: re}
}

type responseProcessor struct {
	conv      DomainConverter
	urlRegexp *regexp.Regexp
}

func (p *responseProcessor) Process(c *fiber.Ctx, resp *http.Response) {
	p.convertCORS(resp)
	p.removeCSP(resp)
	p.convertLocation(resp)
	p.writeCookies(c, resp)
	p.writeHeaders(c, resp)
	p.writeBody(c, resp)
}

func (p *responseProcessor) convertCORS(resp *http.Response) {
	originHeader := resp.Request.Header["Origin"]
	if len(originHeader) > 0 {
		proxyOrigin := p.conv.ToProxy(originHeader[0])
		resp.Header["Access-Control-Allow-Origin"] = []string{proxyOrigin}
	}
}

func (*responseProcessor) removeCSP(resp *http.Response) {
	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("Content-Security-Policy-Report-Only")
}

func (p *responseProcessor) convertLocation(resp *http.Response) {
	p.convertHeaderDomain(resp, "Location")
	p.convertHeaderDomain(resp, "Content-Location")
}

func (p *responseProcessor) convertHeaderDomain(resp *http.Response, headerName string) {
	value := resp.Header[headerName]
	if len(value) == 0 {
		return
	}
	u, err := url.Parse(value[0])
	if err != nil {
		return
	}
	if len(u.Host) > 0 {
		u.Host = p.conv.ToProxy(u.Host)
	}
	resp.Header[headerName] = []string{u.String()}
}

func (p *responseProcessor) writeCookies(c *fiber.Ctx, resp *http.Response) {
	// TODO ToProxy cookie
	for _, cookie := range resp.Cookies() {
		cookie.SameSite = http.SameSiteNoneMode
		cookie.HttpOnly = false
		cookie.Secure = true
		cookie.Domain = p.conv.ToProxy(cookie.Domain)
		if v := cookie.String(); v != "" {
			c.Response().Header.Add("Set-Cookie", v)
		}
	}
	resp.Header.Del("Set-Cookie")
}

func (*responseProcessor) writeHeaders(c *fiber.Ctx, resp *http.Response) {
	for header, values := range resp.Header {
		for _, v := range values {
			c.Response().Header.Add(header, v)
		}
	}
}

// const jsFile = `
// var find = "\\."
// var rep = "-"

// var findUrl = /(-\w*)\//

// function changeUrl(str) {
//     if (str.includes("juicyrout")) {
//         return str;
//     }
//     var replacedStr = str.replace(new RegExp(find, 'g'), rep)
//     var replacedStr1 = replacedStr.replace(findUrl, "$1.host.juicyrout:8091/")
//     return replacedStr1
// }

// var constantMock = window.fetch;
//  window.fetch = function(url, config) {
//     var args = Array.prototype.slice.call(arguments)
//     console.log.apply(console, args)
// 	arguments[0] = changeUrl(arguments[0])
//     return constantMock.apply(this, arguments)
//  }

// let oldXHROpen = window.XMLHttpRequest.prototype.open;
// window.XMLHttpRequest.prototype.open = function(method, url, async, user, password) {
//     arguments[1] = changeUrl(arguments[1])

// 	var args = Array.prototype.slice.call(arguments)
// 	console.log.apply(console, args)
//  this.addEventListener('load', function() {

//   console.log('load: ' + this.responseText)
//  })

//  return oldXHROpen.apply(this, arguments)
// }
// `

// TODO HTML fetch hook
//nolint:errcheck
func (p *responseProcessor) writeBody(w io.Writer, resp *http.Response) {
	// TODO multi content-type response handler
	// contentType := resp.Header["Content-Type"]
	// if len(contentType) > 0 && strings.Contains(contentType[0], "script") {
	// 	// w.Write([]byte(jsFile))
	// }

	var buff bytes.Buffer
	for {
		n, err := buff.ReadFrom(io.LimitReader(resp.Body, 4096))
		bytearr := buff.Bytes()
		foundIndex := p.urlRegexp.FindAllIndex(bytearr, -1)
		start := 0
		for _, pair := range foundIndex {
			w.Write(bytearr[start:pair[0]])

			urlString := string(bytearr[pair[0]:pair[1]])
			w.Write([]byte(p.conv.ToProxy(urlString)))
			start = pair[1]
		}
		// advance the buffer by the number of processed bytes
		if start > 0 {
			buff.Next(start)
			// move buffer from start to 0
			bytearr = buff.Bytes()
			buff.Reset()
			buff.Write(bytearr)
		} else {
			w.Write(buff.Bytes())
			buff.Reset()
		}

		if n == 0 || err != nil {
			if err != nil && err != io.EOF {
				log.Println("io error", err)
			}
			w.Write(buff.Bytes())
			return
		}
	}
}
