/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package routeguide

import (
	"io"
	"math/rand"
	"time"
)

import (
	log "github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func init() {
	config.SetConsumerServiceByInterfaceName("io.grpc.examples.helloworld.GreeterGrpc$RouteGuide", &RouteGuideClientImpl{})
	config.SetConsumerService(&RouteGuideClientImpl{})
}

// printFeatures lists all the features within the given bounding Rectangle.
func PrintFeatures(stream RouteGuide_ListFeaturesClient) {
	for {
		feature, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Fait to receive a feature: %v", err)
		}
		log.Infof("Feature: name: %q, point:(%v, %v)", feature.GetName(),
			feature.GetLocation().GetLatitude(), feature.GetLocation().GetLongitude())
	}
}

// runRecordRoute sends a sequence of points to server and expects to get a RouteSummary from server.
func RunRecordRoute(stream RouteGuide_RecordRouteClient) {
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pointCount := int(r.Int31n(100)) + 2 // Traverse at least two points
	var points []*Point
	for i := 0; i < pointCount; i++ {
		points = append(points, randomPoint(r))
	}
	log.Infof("Traversing %d points.", len(points))
	for _, point := range points {
		if err := stream.Send(point); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, point, err)
		}
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Infof("Route summary: %v", reply)
}

// runRouteChat receives a sequence of route notes, while sending notes for various locations.
func RunRouteChat(stream RouteGuide_RouteChatClient) {
	notes := []*RouteNote{
		{Location: &Point{Latitude: 0, Longitude: 1}, Message: "First message"},
		{Location: &Point{Latitude: 0, Longitude: 2}, Message: "Second message"},
		{Location: &Point{Latitude: 0, Longitude: 3}, Message: "Third message"},
		{Location: &Point{Latitude: 0, Longitude: 1}, Message: "Fourth message"},
		{Location: &Point{Latitude: 0, Longitude: 2}, Message: "Fifth message"},
		{Location: &Point{Latitude: 0, Longitude: 3}, Message: "Sixth message"},
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			log.Infof("Got message %s at point(%d, %d)", in.Message, in.Location.Latitude, in.Location.Longitude)
		}
	}()
	for _, note := range notes {
		if err := stream.Send(note); err != nil {
			log.Fatalf("Failed to send a note: %v", err)
		}
	}
	stream.CloseSend()
	<-waitc
}

func randomPoint(r *rand.Rand) *Point {
	lat := (r.Int31n(180) - 90) * 1e7
	long := (r.Int31n(360) - 180) * 1e7
	return &Point{Latitude: lat, Longitude: long}
}
