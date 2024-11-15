package main

import (
	logog "log"
	"net/http"
	"net/http/pprof"
	"os"

	"cosmossdk.io/log"

	"github.com/CosmWasm/wasmd/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/custom_debug_path/profile", pprof.Profile)
	logog.Fatal(http.ListenAndServe(":6060", mux))
}
