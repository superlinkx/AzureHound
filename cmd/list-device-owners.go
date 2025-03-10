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
	"sync"
	"time"

	"github.com/bloodhoundad/azurehound/client"
	"github.com/bloodhoundad/azurehound/enums"
	"github.com/bloodhoundad/azurehound/models"
	"github.com/bloodhoundad/azurehound/pipeline"
	"github.com/spf13/cobra"
)

func init() {
	listRootCmd.AddCommand(listDeviceOwnersCmd)
}

var listDeviceOwnersCmd = &cobra.Command{
	Use:          "device-owners",
	Long:         "Lists Azure AD Device Owners",
	Run:          listDeviceOwnersCmdImpl,
	SilenceUsage: true,
}

func listDeviceOwnersCmdImpl(cmd *cobra.Command, args []string) {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
	defer gracefulShutdown(stop)

	log.V(1).Info("testing connections")
	if err := testConnections(); err != nil {
		exit(err)
	} else if azClient, err := newAzureClient(); err != nil {
		exit(err)
	} else {
		log.Info("collecting azure device owners...")
		start := time.Now()
		stream := listDeviceOwners(ctx, azClient, listDevices(ctx, azClient))
		outputStream(ctx, stream)
		duration := time.Since(start)
		log.Info("collection completed", "duration", duration.String())
	}
}

func listDeviceOwners(ctx context.Context, client client.AzureClient, devices <-chan interface{}) <-chan interface{} {
	var (
		out     = make(chan interface{})
		ids     = make(chan string)
		streams = pipeline.Demux(ctx.Done(), ids, 25)
		wg      sync.WaitGroup
	)

	go func() {
		defer close(ids)

		for result := range pipeline.OrDone(ctx.Done(), devices) {
			if device, ok := result.(AzureWrapper).Data.(models.Device); !ok {
				log.Error(fmt.Errorf("failed type assertion"), "unable to continue enumerating device owners", "result", result)
			} else {
				ids <- device.Id
			}
		}
	}()

	wg.Add(len(streams))
	for i := range streams {
		stream := streams[i]
		go func() {
			defer wg.Done()
			for id := range stream {
				var (
					data = models.DeviceOwners{
						DeviceId: id,
					}
					count = 0
				)
				for item := range client.ListAzureDeviceRegisteredOwners(ctx, id, false) {
					if item.Error != nil {
						log.Error(item.Error, "unable to continue processing owners for this device", "deviceId", id)
					} else {
						deviceOwner := models.DeviceOwner{
							Owner:    item.Ok,
							DeviceId: item.DeviceId,
						}
						log.V(2).Info("found device owner", "deviceOwner", deviceOwner)
						count++
						data.Owners = append(data.Owners, deviceOwner)
					}
				}
				out <- AzureWrapper{
					Kind: enums.KindAZDeviceOwner,
					Data: data,
				}
				log.V(1).Info("finished listing device owners", "deviceId", id, "count", count)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
		log.Info("finished listing all device owners")
	}()

	return out
}
