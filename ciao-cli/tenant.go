//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/01org/ciao/payloads"
)

var tenantCommand = &command{
	SubCommands: map[string]subCommand{
		"list": new(tenantListCommand),
	},
}

type tenantListCommand struct {
	Flag      flag.FlagSet
	all       bool
	quotas    bool
	resources bool
}

func (cmd *tenantListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] tenant list

List tenants for the current user

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *tenantListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.all, "all", false, "List all tenants")
	cmd.Flag.BoolVar(&cmd.quotas, "quotas", false, "List quotas status for a tenant")
	cmd.Flag.BoolVar(&cmd.resources, "resources", false, "List consumed resources for a tenant for the past 15mn")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantListCommand) run(args []string) error {
	if cmd.all {
		return listAllTenants()
	}
	if cmd.quotas {
		return listTenantQuotas()
	}
	if cmd.resources {
		return listTenantResources()
	}

	return listUserTenants()
}

func listAllTenants() error {
	projects, err := getAllProjects(*identityUser, *identityPassword)
	if err != nil {
		fatalf(err.Error())
	}
	for i, project := range projects.Projects {
		fmt.Printf("Tenant [%d]\n", i+1)
		fmt.Printf("\tUUID: %s\n", project.ID)
		fmt.Printf("\tName: %s\n", project.Name)
	}
	return nil
}

func listUserTenants() error {
	projects, err := getUserProjects(*identityUser, *identityPassword)
	if err != nil {
		fatalf(err.Error())
	}
	fmt.Printf("Projects for user %s\n", *identityUser)
	for _, project := range projects {
		fmt.Printf("\tUUID: %s\n", project.ID)
		fmt.Printf("\tName: %s\n", project.Name)
	}
	return nil
}

func listTenantQuotas() error {
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var resources payloads.CiaoTenantResources
	url := buildComputeURL("%s/quotas", *tenantID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &resources)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("Quotas for tenant %s:\n", resources.ID)
	fmt.Printf("\tInstances: %d | %s\n", resources.InstanceUsage, limitToString(resources.InstanceLimit))
	fmt.Printf("\tCPUs:      %d | %s\n", resources.VCPUUsage, limitToString(resources.VCPULimit))
	fmt.Printf("\tMemory:    %d | %s\n", resources.MemUsage, limitToString(resources.MemLimit))
	fmt.Printf("\tDisk:      %d | %s\n", resources.DiskUsage, limitToString(resources.DiskLimit))

	return nil
}

func listTenantResources() error {
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var usage payloads.CiaoUsageHistory
	url := buildComputeURL("%s/resources", *tenantID)

	now := time.Now()
	values := []queryValue{
		{
			name:  "start_date",
			value: now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			name:  "end_date",
			value: now.Format(time.RFC3339),
		},
	}

	resp, err := sendHTTPRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &usage)
	if err != nil {
		fatalf(err.Error())
	}

	if len(usage.Usages) == 0 {
		fmt.Printf("No usage history for %s\n", *tenantID)
		return nil
	}

	fmt.Printf("Usage for tenant %s:\n", *tenantID)
	for _, u := range usage.Usages {
		fmt.Printf("\t%v: [%d CPUs] [%d MB memory] [%d MB disk]\n", u.Timestamp, u.VCPU, u.Memory, u.Disk)
	}

	return nil
}
