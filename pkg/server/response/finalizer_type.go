package response

type RedirectFinalizer struct {
	Code int
	URL  string
}

func Redirect(code int, url string) *RedirectFinalizer {
	return &RedirectFinalizer{Code: code, URL: url}
}

//

type NoContentFinalizer struct {
	Code int
}

func NoContent(code int) *NoContentFinalizer {
	return &NoContentFinalizer{Code: code}
}

//

type JSONFinalizer struct {
	Code int
	JSON interface{}
}

func JSON(code int, json interface{}) *JSONFinalizer {
	return &JSONFinalizer{Code: code, JSON: json}
}

//

type HTMLFinalizer struct {
	Code int
	HTML string
}

func HTML(code int, html string) *HTMLFinalizer {
	return &HTMLFinalizer{Code: code, HTML: html}
}

//

type HTMLBlobFinalizer struct {
	Code int
	HTML []byte
}

func HTMLBlob(code int, html []byte) *HTMLBlobFinalizer {
	return &HTMLBlobFinalizer{Code: code, HTML: html}
}

//

type StringFinalizer struct {
	Code   int
	String string
}

func String(code int, s string) *StringFinalizer {
	return &StringFinalizer{Code: code, String: s}
}
