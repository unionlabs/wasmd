package main

import (
	logog "log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"cosmossdk.io/log"

	"github.com/CosmWasm/wasmd/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	go func() {
		logog.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	rootCmd := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
