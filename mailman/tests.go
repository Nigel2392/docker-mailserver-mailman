package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
)

var outFilePath = "./out.log"
var outFile *os.File

func init() {
	var err error
	outFile, err = os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
}

func writeCommandOutputFile(cmd *mailmgmt.Command) {
	stdOut, stdErr, err := cmd.Exec()
	outFile.Write([]byte(cmd.String()))
	outFile.Write([]byte("\n"))

	if err != nil {
		outFile.Write([]byte("error:\n"))
		outFile.Write([]byte(err.Error()))
		outFile.Write([]byte("\n"))
	}

	if len(stdOut) > 0 {
		outFile.Write([]byte("stdout:\n"))
		outFile.Write([]byte(stdOut))
		outFile.Write([]byte("\n"))
	}

	if len(stdErr) > 0 {
		outFile.Write([]byte("stderr:\n"))
		outFile.Write([]byte(stdErr))
		outFile.Write([]byte("\n"))
	}

	outFile.Write([]byte("------------------------------------------------------------------\n"))
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func testOutputs() {
	mails := must(mailmgmt.CONFIG.CommandSetup(context.Background()).Email().List())
	for _, addr := range mails {
		fmt.Fprintf(outFile, "Address: %s\n", addr.Email)
		fmt.Fprintf(outFile, "Current Quota: %d\n", addr.CurrentQuota)
		fmt.Fprintf(outFile, "Max Quota: %d\n", addr.MaxQuota)
		fmt.Fprintf(outFile, "Percentage Full: %d%%\n", addr.PercentageFull)

		fmt.Fprintln(outFile, "Aliases:")

		for _, alias := range addr.Aliases {
			fmt.Fprintf(outFile, "- %s\n", alias)
		}
	}

	aliases := must(mailmgmt.CONFIG.CommandSetup(context.Background()).Alias().Map())
	for target, list := range aliases {
		fmt.Fprintf(outFile, "%s:\n", target)
		for _, list := range list {
			fmt.Fprintf(outFile, "- %s\n", list)
		}
	}

	sendRejected := must(mailmgmt.Setup().Restrict().List().Send())
	for _, addr := range sendRejected {
		fmt.Fprintf(outFile, "Address: %s\n", addr.Address)
		fmt.Fprintf(outFile, "Reason: %q\n", addr.Status)
	}

	recvRejected := must(mailmgmt.Setup().Restrict().List().Receive())
	for _, list := range recvRejected {
		fmt.Fprintf(outFile, "Address: %s\n", list.Address)
		fmt.Fprintf(outFile, "Reason: %q\n", list.Status)

	}
}

func testCommands() {
	testOutputs()

	//var cmd *mailmgmt.Command
	//
	//mails, _ := mailmgmt.CONFIG.CommandSetup(context.Background()).Email().List()
	//for _, addr := range mails {
	//	mailmgmt.CONFIG.CommandSetup(context.Background()).Email().Delete(addr.Email)
	//}
	//
	//for i := range 14 {
	//	cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Email().CommandAdd(fmt.Sprintf("test%d@go-dev.nl", i), "test")
	//	writeCommandOutputFile(cmd)
	//
	//	for j := range 3 {
	//		cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Alias().CommandAdd(fmt.Sprintf("test%d-%d@go-dev.nl", i, j), fmt.Sprintf("test%d@go-dev.nl", i))
	//		writeCommandOutputFile(cmd)
	//	}
	//}
	//
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Email().CommandUpdate("test1@go-dev.nl", "test1")
	//writeCommandOutputFile(cmd)
	//
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Email().CommandList()
	//writeCommandOutputFile(cmd)
	//
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Email().CommandList()
	//writeCommandOutputFile(cmd)
	//
	//// List all send restrictions (restrict list send)
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().List().CommandSend()
	//writeCommandOutputFile(cmd)
	//
	//// List all receive restrictions (restrict list receive)
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().List().CommandReceive()
	//writeCommandOutputFile(cmd)
	//
	//// Add send restriction for test1@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandSend("test1@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Add send restriction for test2@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandSend("test2@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Add send restriction for test3@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandSend("test3@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Add receive restriction for test1@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandReceive("test1@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Add receive restriction for test2@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandReceive("test2@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Add receive restriction for test3@go-dev.nl
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().Add().CommandReceive("test3@go-dev.nl")
	//writeCommandOutputFile(cmd)
	//
	//// Verify send restrictions were added by listing again
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().List().CommandSend()
	//writeCommandOutputFile(cmd)
	//
	//// Verify receive restrictions were added by listing again
	//cmd = mailmgmt.CONFIG.CommandSetup(context.Background()).Restrict().List().CommandReceive()
	//writeCommandOutputFile(cmd)
}
