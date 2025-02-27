// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bloodhoundad/azurehound/client"
	"github.com/bloodhoundad/azurehound/config"
	"github.com/bloodhoundad/azurehound/enums"
	kinds "github.com/bloodhoundad/azurehound/enums"
	"github.com/bloodhoundad/azurehound/models"
	"github.com/bloodhoundad/azurehound/pipeline"
	"github.com/spf13/cobra"
)

var listKeyVaultAccessPoliciesCmd = &cobra.Command{
	Use:          "key-vault-access-policies",
	Long:         "Lists Azure Key Vault Access Policies",
	Run:          listKeyVaultAccessPoliciesCmdImpl,
	SilenceUsage: true,
}

func init() {
	config.Init(listKeyVaultAccessPoliciesCmd, []config.Config{config.KeyVaultAccessTypes})
	listRootCmd.AddCommand(listKeyVaultAccessPoliciesCmd)
}

func listKeyVaultAccessPoliciesCmdImpl(cmd *cobra.Command, args []string) {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
	defer gracefulShutdown(stop)

	log.V(1).Info("testing connections")
	if err := testConnections(); err != nil {
		exit(err)
	} else if azClient, err := newAzureClient(); err != nil {
		exit(err)
	} else {
		log.Info("collecting azure key vault access policies...")
		start := time.Now()
		subscriptions := listSubscriptions(ctx, azClient)
		if filters, ok := config.KeyVaultAccessTypes.Value().([]enums.KeyVaultAccessType); !ok {
			exit(fmt.Errorf("filter failed type assertion"))
		} else {
			if len(filters) > 0 {
				log.Info("applying access type filters", "filters", filters)
			}
			stream := listKeyVaultAccessPolicies(ctx, azClient, listKeyVaults(ctx, azClient, subscriptions), filters)
			outputStream(ctx, stream)
			duration := time.Since(start)
			log.Info("collection completed", "duration", duration.String())
		}
	}
}

func listKeyVaultAccessPolicies(ctx context.Context, client client.AzureClient, keyVaults <-chan interface{}, filters []enums.KeyVaultAccessType) <-chan interface{} {
	out := make(chan interface{})

	go func() {
		defer close(out)

		for result := range pipeline.OrDone(ctx.Done(), keyVaults) {
			if keyVault, ok := result.(AzureWrapper).Data.(models.KeyVault); !ok {
				log.Error(fmt.Errorf("failed type assertion"), "unable to continue enumerating key vault access policies", "result", result)
				return
			} else {
				for _, policy := range keyVault.Properties.AccessPolicies {
					if len(filters) == 0 {
						out <- AzureWrapper{
							Kind: kinds.KindAZKeyVaultAccessPolicy,
							Data: models.KeyVaultAccessPolicy{
								KeyVaultId:        keyVault.Id,
								AccessPolicyEntry: policy,
							},
						}
					} else {
						for _, filter := range filters {
							permissions := func() []string {
								switch filter {
								case enums.GetCerts:
									return policy.Permissions.Certificates
								case enums.GetKeys:
									return policy.Permissions.Keys
								case enums.GetSecrets:
									return policy.Permissions.Secrets
								default:
									log.Error(fmt.Errorf("unsupported key vault access type: %s", filter), "unable to apply key vault access policy filter")
									return []string{}
								}
							}()
							if contains(permissions, "Get") {
								out <- AzureWrapper{
									Kind: kinds.KindAZKeyVaultAccessPolicy,
									Data: models.KeyVaultAccessPolicy{
										KeyVaultId:        keyVault.Id,
										AccessPolicyEntry: policy,
									},
								}
								break
							}
						}
					}
				}
			}
		}
	}()

	return out
}
