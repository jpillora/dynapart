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

var version = "0.0.0-src"

type cli struct {
	MaxPages       int      `opts:"help=maximum number of result patches to return"`
	ConsistentRead bool     `opts:"help=enable consistent reads"`
	ItemFormat     string   `opts:"help=json is the only format currently"`
	NoItemNumber   bool     `opts:"short=n, help=hide item numbers"`
	NoColors       bool     `opts:"short=c, help=disable json syntax highlighting"`
	Statement      string   `opts:"mode=arg"`
	Args           []string `opts:"mode=arg"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	//execute cli
	c := cli{
		MaxPages: 1,
		NoColors: !isatty.IsTerminal(os.Stdout.Fd()),
	}
	opts.
		New(&c).
		Version(version).
		DocAfter("usage", "more", "\nExecutes the given DynamoDB PartiQL <statement>,\n"+
			"which may contain ? placeholders. If placeholders are set,\n"+
			"you must provide corresponding [arg]s where each [arg] is valid JSON.\n").
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
		return db.ListTablesPagesWithContext(ctx, &dynamodb.ListTablesInput{}, func(lto *dynamodb.ListTablesOutput, b bool) bool {
			for _, t := range lto.TableNames {
				fmt.Println(*t)
			}
			return true
		})
	}
	//
	printer := printer{colors: !c.NoColors, nums: !c.NoItemNumber}
	//execute
	var next *string
	page := 1
	t0 := time.Now()
	for {
		out, err := db.ExecuteStatementWithContext(ctx, &dynamodb.ExecuteStatementInput{
			Statement:  &c.Statement,
			Parameters: params,
			NextToken:  next,
		})
		if err != nil {
			return err
		}
		printer.items(out.Items)
		if out.NextToken == nil || page == c.MaxPages {
			break
		}
		next = out.NextToken
		page++
	}
	fmt.Printf("[returned %d items in %s]\n", printer.n, time.Since(t0))
	return nil
}

type attr = dynamodb.AttributeValue

type attrs = []*attr

type item = map[string]*attr

type items = []item

type printer struct {
	n      int
	nums   bool
	colors bool
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
	if p.nums {
		fmt.Printf("[#%d] ", p.n)
	}
	fmt.Println(s)
}

func interruptibleContext() context.Context {
	//allow user to gracefully shutdown with Ctrl+C
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
