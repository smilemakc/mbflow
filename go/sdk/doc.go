// Package mbflow provides a Go SDK for the MBFlow workflow engine.
//
// The SDK supports two transport modes:
//   - HTTP: communicates with MBFlow server via REST API
//   - gRPC: communicates via gRPC Service API (preferred for performance)
//
// Quick start:
//
//	client, err := mbflow.NewClient(
//	    mbflow.WithHTTP("http://localhost:8585"),
//	    mbflow.WithAPIKey("your-api-key"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	workflows, err := client.Workflows().List(ctx, nil)
package mbflow