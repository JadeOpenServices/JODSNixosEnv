package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/nettrust"
)

const vpnStatusPath = "/run/gjallar/vpn-trust.json"

func runVPN(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gjallarctl vpn apply --policy FILE | status")
		return 2
	}
	switch args[0] {
	case "apply":
		return runVPNApply(args[1:], stdout, stderr)
	case "status":
		return runVPNStatus(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "ERROR: unknown vpn command %q\n", args[0])
		return 2
	}
}

func runVPNApply(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vpn apply", flag.ContinueOnError)
	flags.SetOutput(stderr)
	policyPath := flags.String("policy", "", "network trust policy JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *policyPath == "" {
		fmt.Fprintln(stderr, "ERROR: --policy is required")
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(stderr, "ERROR: gjallarctl vpn apply changes routing and must run as root")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	policy, err := nettrust.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	sys := nettrust.NewSystem()
	network, err := sys.Network(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: observe network: %v\n", err)
		return 1
	}
	tailnet, err := sys.Tailnet(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: observe tailnet: %v\n", err)
		return 1
	}

	verdict, err := nettrust.Decide(ctx, policy, network, tailnet, sys)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	for _, warning := range verdict.Warnings {
		fmt.Fprintf(stderr, "WARN: %s\n", warning)
	}

	code := 0
	if err := sys.ApplyBypass(ctx, verdict.Bypass); err != nil {
		fmt.Fprintf(stderr, "ERROR: apply LAN bypass: %v\n", err)
		code = 1
	}
	if verdict.ManageExitNode {
		changed, err := sys.ApplyExitNode(ctx, tailnet, verdict.ExitNode)
		switch {
		case err != nil:
			fmt.Fprintf(stderr, "ERROR: set exit node: %v\n", err)
			code = 1
		case changed && verdict.ExitNode == "":
			fmt.Fprintln(stdout, "VPN: exit node off")
		case changed:
			fmt.Fprintf(stdout, "VPN: exit node %s on\n", verdict.ExitNode)
		}
	}

	fmt.Fprintf(stdout, "network %s: %s; bypass [%s]\n",
		verdict.Trust, verdict.Reason, strings.Join(verdict.Bypass, " "))

	if err := nettrust.WriteStatus(vpnStatusPath, verdict); err != nil {
		fmt.Fprintf(stderr, "WARN: write %s: %v\n", vpnStatusPath, err)
	}
	return code
}

func runVPNStatus(stdout, stderr io.Writer) int {
	data, err := os.ReadFile(vpnStatusPath)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: no network trust decision yet: %v\n", err)
		return 1
	}
	var v nettrust.Verdict
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Fprintf(stderr, "ERROR: parse %s: %v\n", vpnStatusPath, err)
		return 1
	}
	fmt.Fprintf(stdout, "network:   %s\nreason:    %s\n", v.Trust, v.Reason)
	if v.SSID != "" {
		fmt.Fprintf(stdout, "wifi:      %s\n", v.SSID)
	}
	if v.Router != "" {
		fmt.Fprintf(stdout, "router:    %s\n", v.Router)
	}
	exit := "unmanaged"
	if v.ManageExitNode {
		exit = "off"
		if v.ExitNode != "" {
			exit = v.ExitNode
		}
	}
	fmt.Fprintf(stdout, "exit node: %s\nlocal:     %s\n", exit, strings.Join(v.Bypass, " "))
	for _, w := range v.Warnings {
		fmt.Fprintf(stdout, "warning:   %s\n", w)
	}
	return 0
}
