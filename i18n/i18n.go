package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v2"
)

var p *message.Printer
var lock sync.Mutex

// PluralRule is Plural rule
type PluralRule struct {
	Pos   int
	Expr  string
	Value int
	Text  string
}

// Message is translation message
type Message map[string]string

func init() {
	// default use English
	p = message.NewPrinter(language.English)
}

// SetLang set language
func SetLang(lang language.Tag) {
	lock.Lock()
	defer lock.Unlock()
	p = message.NewPrinter(lang)
}

// Printf is like fmt.Printf, but using language-specific formatting.
func Printf(format string, args ...interface{}) {
	format, args = preArgs(format, args...)
	p.Printf(format, args...)
}

// Sprintf is like fmt.Sprintf, but using language-specific formatting.
func Sprintf(format string, args ...interface{}) string {
	format, args = preArgs(format, args...)
	return p.Sprintf(format, args...)
}

// Fprintf is like fmt.Fprintf, but using language-specific formatting.
func Fprintf(w io.Writer, key message.Reference, a ...interface{}) (n int, err error) {
	format, args := preArgs(key.(string), a...)
	key = message.Reference(format)
	return p.Fprintf(w, key, args...)
}

// Preprocessing parameters in plural form
func preArgs(format string, args ...interface{}) (string, []interface{}) {
	length := len(args)
	if length > 0 {
		lastArg := args[length-1]
		switch lastArg.(type) {
		case []PluralRule:
			rules := lastArg.([]PluralRule)
			// parse rule
			for _, rule := range rules {
				curPosVal := args[rule.Pos-1].(int)
				// Support comparison expression
				if (rule.Expr == "=" && curPosVal == rule.Value) || (rule.Expr == ">" && curPosVal > rule.Value) {
					format = rule.Text
					break
				}
			}
			args = args[0:strings.Count(format, "%")]
		}
	}
	return format, args
}

// Plural is Plural function
func Plural(cases ...interface{}) []PluralRule {
	rules := []PluralRule{}
	// %[1]d=1, %[1]d>1
	re := regexp.MustCompile(`\[(\d+)\][^=>]\s*(\=|\>)\s*(\d+)$`)
	for i := 0; i < len(cases); {
		expr := cases[i].(string)
		if i++; i >= len(cases) {
			return rules
		}
		text := cases[i].(string)
		// cannot match continue
		if !re.MatchString(expr) {
			continue
		}
		matches := re.FindStringSubmatch(expr)
		pos, _ := strconv.Atoi(matches[1])
		value, _ := strconv.Atoi(matches[3])
		rules = append(rules, PluralRule{
			Pos:   pos,
			Expr:  matches[2],
			Value: value,
			Text:  text,
		})
		i++
	}
	return rules
}

func unmarshal(path string) (*Message, error) {
	result := &Message{}
	fileExt := strings.ToLower(filepath.Ext(path))
	if fileExt != ".toml" && fileExt != ".json" && fileExt != ".yaml" {
		return result, fmt.Errorf(Sprintf("File type not supported"))
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return result, nil
	}

	if strings.HasSuffix(fileExt, ".json") {
		err := json.Unmarshal(buf, result)
		if err != nil {
			return result, err
		}
	}

	if strings.HasSuffix(fileExt, ".yaml") {
		err := yaml.Unmarshal(buf, result)
		if err != nil {
			return result, err
		}
	}

	if strings.HasSuffix(fileExt, ".toml") {
		_, err := toml.Decode(string(buf), result)
		if err != nil {
			return result, err
		}
	}
	return result, nil

}

func marshal(v interface{}, format string) ([]byte, error) {
	switch format {
	case "json":
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(v)
		return buffer.Bytes(), err
	case "toml":
		var buf bytes.Buffer
		enc := toml.NewEncoder(&buf)
		enc.Indent = ""
		err := enc.Encode(v)
		return buf.Bytes(), err
	case "yaml":
		return yaml.Marshal(v)
	}
	return nil, fmt.Errorf("unsupported format: %s", format)
}
