package main

import (
	"encoding/base64"
	"image/color"
	"syscall/js"

	"github.com/caiguanhao/qrcode"
)

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("QRCode", js.FuncOf(QRCode))
	js.Global().Call("eval", "document.dispatchEvent(new Event('QRCode.Ready'))")
	<-c
}

func QRCode(this js.Value, inputs []js.Value) interface{} {
	window := js.Global()
	return window.Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve, reject := args[0], args[1]
		go func() {
			res, err := newQRCode(inputs)
			if e, ok := err.(error); ok {
				reject.Invoke(window.Get("Error").New(e.Error()))
			} else if e, ok := err.(string); ok {
				reject.Invoke(window.Get("Error").New(e))
			} else if err != nil {
				reject.Invoke(window.Get("Error").New("unknown error"))
			} else {
				resolve.Invoke(res)
			}
		}()
		return nil
	}))
}

func newQRCode(inputs []js.Value) (interface{}, interface{}) {
	var content string
	var lvl qrcode.RecoveryLevel = qrcode.Medium
	var size int = 300
	var format string = "object-url"
	var padding int = 0
	var backgroundColor color.RGBA = color.RGBA{0xff, 0xff, 0xff, 0xff}
	var foregroundColor color.RGBA = color.RGBA{0, 0, 0, 0xff}
	if len(inputs) > 0 && inputs[0].Type() == js.TypeObject {
		opts := inputs[0]
		if l := opts.Get("level"); l.Type() != js.TypeUndefined {
			switch l.String() {
			case "low":
				lvl = qrcode.Low
			case "medium":
				lvl = qrcode.Medium
			case "high":
				lvl = qrcode.High
			case "highest":
				lvl = qrcode.Highest
			default:
				return nil, "unknown level"
			}
		}
		if s := opts.Get("size"); s.Type() != js.TypeUndefined {
			if s.Type() == js.TypeNumber {
				size = s.Int()
			} else {
				return nil, "size must be a number"
			}
		}
		if p := opts.Get("padding"); p.Type() != js.TypeUndefined {
			if p.Type() == js.TypeNumber {
				padding = p.Int()
				if padding < 0 {
					return nil, "padding must not less than 0"
				}
			} else {
				return nil, "padding must be a number"
			}
		}
		content = opts.Get("content").String()
		if f := opts.Get("format"); f.Type() != js.TypeUndefined {
			format = f.String()
		}
		if bgColor := opts.Get("backgroundColor"); bgColor.Type() != js.TypeUndefined {
			if !setColorWithHex(&backgroundColor, bgColor.String()) {
				return nil, "invalid background color hex string"
			}
		}
		if fgColor := opts.Get("foregroundColor"); fgColor.Type() != js.TypeUndefined {
			if !setColorWithHex(&foregroundColor, fgColor.String()) {
				return nil, "invalid foreground color hex string"
			}
		}
	} else if len(inputs) > 0 && inputs[0].Type() == js.TypeString {
		content = inputs[0].String()
	}
	q, err := qrcode.New(content, lvl)
	if err != nil {
		return nil, err
	}
	q.QuietZoneSize = padding
	q.BackgroundColor = backgroundColor
	q.ForegroundColor = foregroundColor
	if format == "string" {
		return q.ToString(false), nil
	}
	if format == "string-inverted" {
		return q.ToString(true), nil
	}
	if format == "small-string" {
		return q.ToSmallString(false), nil
	}
	if format == "small-string-inverted" {
		return q.ToSmallString(true), nil
	}
	if format == "svg" {
		data, err := q.SVG(size)
		if err != nil {
			return nil, err
		}
		return string(data), nil
	}
	if format == "base64" || format == "uint8array" || format == "blob" || format == "object-url" {
		data, err := q.PNG(size)
		if err != nil {
			return nil, err
		}
		if format == "base64" {
			return base64.StdEncoding.EncodeToString(data), nil
		}
		window := js.Global()
		uint8array := window.Get("Uint8Array").New(len(data))
		js.CopyBytesToJS(uint8array, data)
		if format == "uint8array" {
			return uint8array, nil
		}
		opt := window.Get("Object").New()
		opt.Set("type", "image/png")
		blob := window.Get("Blob").New(window.Get("Array").Call("of", uint8array), opt)
		if format == "blob" {
			return blob, nil
		}
		url := window.Get("URL").Call("createObjectURL", blob)
		if format == "object-url" {
			return url, nil
		}
	}
	return nil, "unknown format"
}

func setColorWithHex(c *color.RGBA, hex string) bool {
	if (len(hex) == 6 || len(hex) == 3) && hex[0] != '#' {
		hex = "#" + hex
	}
	c.A = 0xff
	switch len(hex) {
	case 7:
		c.R = hexToByte(hex[1])<<4 + hexToByte(hex[2])
		c.G = hexToByte(hex[3])<<4 + hexToByte(hex[4])
		c.B = hexToByte(hex[5])<<4 + hexToByte(hex[6])
		return true
	case 4:
		c.R = hexToByte(hex[1]) * 17
		c.G = hexToByte(hex[2]) * 17
		c.B = hexToByte(hex[3]) * 17
		return true
	}
	return false
}

func hexToByte(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}
