// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snippets

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/GoogleCloudPlatform/golang-samples/internal/testutil"
	"google.golang.org/protobuf/proto"
)

func createTemplate(ctx context.Context, project, templateName string) error {
	client, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		return err
	}

	disk := &computepb.AttachedDisk{
		AutoDelete: proto.Bool(true),
		Boot:       proto.Bool(true),
		InitializeParams: &computepb.AttachedDiskInitializeParams{
			SourceImage: proto.String("projects/debian-cloud/global/images/family/debian-12"),
			DiskSizeGb:  proto.Int64(25),
			DiskType:    proto.String("pd-balanced"),
		},
	}
	req := &computepb.InsertInstanceTemplateRequest{
		Project: project,
		InstanceTemplateResource: &computepb.InstanceTemplate{
			Name: proto.String(templateName),
			Properties: &computepb.InstanceProperties{
				MachineType: proto.String("n1-standard-4"),
				Disks:       []*computepb.AttachedDisk{disk},
				NetworkInterfaces: []*computepb.NetworkInterface{{
					Name: proto.String("global/networks/default"),
				}},
			},
		},
	}
	op, err := client.Insert(ctx, req)
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func getTemplate(ctx context.Context, project, templateName string) (*computepb.InstanceTemplate, error) {
	client, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	req := &computepb.GetInstanceTemplateRequest{
		Project:          project,
		InstanceTemplate: templateName,
	}
	return client.Get(ctx, req)
}

func deleteTemplate(ctx context.Context, project, templateName string) error {
	client, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		return err
	}

	req := &computepb.DeleteInstanceTemplateRequest{
		Project:          project,
		InstanceTemplate: templateName,
	}
	op, err := client.Delete(ctx, req)
	if err != nil {
		return err
	}

	return op.Wait(ctx)
}

func TestReservations(t *testing.T) {
	var r *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	tc := testutil.SystemTest(t)
	zone := "europe-west2-b"
	reservationName := fmt.Sprintf("test-reservation-%v-%v", time.Now().Format("01-02-2006"), r.Int())
	templateName := fmt.Sprintf("test-template-%v-%v", time.Now().Format("01-02-2006"), r.Int())

	var buf bytes.Buffer
	ctx := context.Background()

	err := createTemplate(ctx, tc.ProjectID, templateName)
	if err != nil {
		t.Errorf("createTemplate got err: %v", err)
	}
	defer deleteTemplate(ctx, tc.ProjectID, templateName)

	sourceTemplate, err := getTemplate(ctx, tc.ProjectID, templateName)
	if err != nil {
		t.Errorf("getTemplate got err: %v", err)
	}

	want := "Reservation created"
	if err := createReservation(&buf, tc.ProjectID, zone, reservationName, *sourceTemplate.SelfLink); err != nil {
		t.Errorf("createReservation got err: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, want) {
		t.Errorf("createReservation got %s, want %s", got, want)
	}
	buf.Reset()

	want = fmt.Sprintf("Reservation: %s", reservationName)
	if err := getReservation(&buf, tc.ProjectID, zone, reservationName); err != nil {
		t.Errorf("getReservation got err: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, want) {
		t.Errorf("getReservation got %s, want %s", got, want)
	}
	buf.Reset()

	want = fmt.Sprintf("- %s %d", reservationName, 2)
	if err := listReservations(&buf, tc.ProjectID, zone); err != nil {
		t.Errorf("listReservations got err: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, want) {
		t.Errorf("listReservations got %s, want %s", got, want)
	}
	buf.Reset()

	want = "Reservation deleted"
	if err := deleteReservation(&buf, tc.ProjectID, zone, reservationName); err != nil {
		t.Errorf("deleteReservation got err: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, want) {
		t.Errorf("deleteReservation got %s, want %s", got, want)
	}
}
