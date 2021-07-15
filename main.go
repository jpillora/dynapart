package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/jpillora/opts"
	"github.com/mattn/go-isatty"
)

const summary = `
Executes the given DynamoDB PartiQL <statement>,
which may contain ? placeholders. If placeholders are set,
you must provide corresponding [arg]s where each [arg] is valid JSON.
`

const extra = `
For more information on using PartiQL with DynamoDB, see
• https://partiql.org/tutorial.html
• https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ql-reference.html

Each DynamoDB item is written to stdout as JSON, one object per line, which makes
dynapart jq compatible.

Syntax highlighting is automatically enabled when stdout is a TTY.
This can be overridden with --no-colors=true/false.

The default AWS SDK authorization process is performed on program start.
Basically, AWS environment variables will be used, followed by a profile (AWS_PROFILE),
followed by the EC2 metadata endpoint. See AWS documentation for details.

You can use a special statement "SHOW TABLES" to list your DynamoDB tables.
Note, This is not valid PartiQL.
`

var version = "0.0.0-src"

type cli struct {
	MaxPages       int      `opts:"help=maximum number of result pages to return"`
	ConsistentRead bool     `opts:"help=enable consistent reads"`
	ItemFormat     string   `opts:"help=json is the only format currently"`
	NoColors       bool     `opts:"short=c, help=disable json syntax highlighting"`
	Verbose        bool     `opts:"help=print actions to stderr"`
	Statement      string   `opts:"mode=arg"`
	Args           []string `opts:"mode=arg"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	plain := !isatty.IsTerminal(os.Stdout.Fd())
	//execute cli
	c := cli{
		MaxPages: 1,
		NoColors: plain,
	}
	opts.
		New(&c).
		Name("dynapart").
		Version(version).
		DocAfter("usage", "more", summary).
		DocBefore("version", "extra", extra).
		Repo("https://github.com/jpillora/dynapart").
		Parse()
	//convert args to json then to dynamodb params
	var params attrs
	for i, arg := range c.Args {
		var v interface{}
		if err := json.Unmarshal([]byte(arg), &v); err != nil {
			return fmt.Errorf("invalid arg '%s' #%d: %s", arg, i, err)
		}
		attr, err := dynamodbattribute.Marshal(v)
		if err != nil {
			return fmt.Errorf("invalid arg '%s' #%d: %s", arg, i, err)
		}
		params = append(params, attr)
	}
	//setup printer
	p := printer{
		colors: !c.NoColors,
		verb:   c.Verbose,
	}
	//setup aws session
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	//setup dynamodb
	db := dynamodb.New(sess)
	//ctrl+c bound context
	ctx := interruptibleContext()
	//magic string "SHOW TABLES"
	if strings.ToUpper(c.Statement) == "SHOW TABLES" {
		p.verbf("execute list tables...\n")
		return db.ListTablesPagesWithContext(ctx, &dynamodb.ListTablesInput{}, func(lto *dynamodb.ListTablesOutput, b bool) bool {
			for _, t := range lto.TableNames {
				fmt.Println(*t)
			}
			return true
		})
	}
	//execute
	var next *string
	page := 1
	t0 := time.Now()
	for {
		p.verbf("execute statement: %s\n", c.Statement)
		out, err := db.ExecuteStatementWithContext(ctx, &dynamodb.ExecuteStatementInput{
			Statement:  &c.Statement,
			Parameters: params,
			NextToken:  next,
		})
		if err != nil {
			return err
		}
		p.items(out.Items)
		if out.NextToken == nil {
			p.verbf("no more items\n")
			break
		}
		if page == c.MaxPages {
			p.verbf("hit max page %d\n", page)
			break
		}
		p.verbf("page %d contained %d items, has more items...", page, len(out.Items))
		next = out.NextToken
		page++
	}
	p.verbf("returned %d items in %s]\n", p.n, time.Since(t0))
	return nil
}

type attr = dynamodb.AttributeValue

type attrs = []*attr

type item = map[string]*attr

type items = []item

type printer struct {
	n            int
	verb, colors bool
}

func (p *printer) items(items items) {
	for _, item := range items {
		p.item(item)
	}
}

func (p *printer) item(item item) {
	var v interface{}
	dynamodbattribute.UnmarshalMap(item, &v)
	j, _ := json.Marshal(v)
	s := string(j)
	if p.colors {
		s = highlight(s)
	}
	p.n++
	if p.verb {
		fmt.Fprintf(os.Stderr, "[#%d] ", p.n)
	}
	fmt.Println(s)
}

func (p *printer) verbf(format string, args ...interface{}) {
	if p.verb {
		fmt.Fprintf(os.Stderr, ">>> "+format, args...)
	}
}

func interruptibleContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		count := 0
		sig := make(chan os.Signal, 1)
		for {
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			s := <-sig
			count++
			if count == 1 {
				log.Printf("user issued %s", s.String())
				cancel()
			} else if count >= 2 {
				log.Printf("user issued %s, force killing", s.String())
				os.Exit(1)
			}
		}
	}()
	return ctx
}

func highlight(json string) string {
	l := lexers.Get("json")
	l = chroma.Coalesce(l)
	f := formatters.TTY256
	s := styles.Get("native")
	it, err := l.Tokenise(nil, json)
	if err != nil {
		panic(err)
	}
	sb := strings.Builder{}
	f.Format(&sb, s, it)
	return sb.String()
}
