package main

import (
	"flag"
	"log"
	loggersub "memo_sample/adapter/logger"
	"memo_sample/di"
	"memo_sample/infra/database"
	"net/http"
	//"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	//"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	//"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	ping := flag.Bool("ping", false, "check ping")
	flag.Parse()

	defer func() {
		_ = (*database.GetDBM()).CloseDB()
	}()

	err := (*database.GetDBM()).ConnectDB()
	if err != nil {
		loggersub.NewLogger().Errorf("db open error: %#+v\n", err)
		return
	}

	interceptor := func(h func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request)	{
			var err error
			if *ping {
				err = (*database.GetDBM()).PingDB()
			}
			if err != nil {
				loggersub.NewLogger().Errorf("db open error: %#+v\n", err)
				panic(err)
			}
			h(w, r)
		}
	}

	//tracer.Start(tracer.WithAgentAddr("host:port"))
	//tracer.Start(tracer.WithAgentAddr("localhost:8080"))
	tracer.Start()
	defer tracer.Stop()

	wrappedHandler := func(pattern string, handler http.HandlerFunc) {
		http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			span := tracer.StartSpan(pattern)
			defer span.Finish()
			//log.Printf("execute http.Handle url:%s", pattern)
			handler(w, r)
		})
	}

	loggersub.NewLogger().Debugf("main called. ping check:%v\n", *ping)


	api := di.InjectAPIServer()
	wrappedHandler("/", interceptor(api.GetMemos))
	wrappedHandler("/post", interceptor(api.PostMemo))
	wrappedHandler("/post/memo_tags", interceptor(api.PostMemoAndTags))
	wrappedHandler("/search/tags_memos", interceptor(api.SearchTagsAndMemos))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		loggersub.NewLogger().Errorf("ListenAndServe error: %#+v\n", err)
	}
}
