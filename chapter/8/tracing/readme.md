# Tracing with OpenTelemetry and Jaeger

TODO: fill in the walk through

## Running this example
- `docker-compose up -d`
- Once started the client application will periodically send requests to the server. Distributed traces will be collected for the requests and responses, then exported for analysis in Jaeger. To view the traces in Jaeger, open http://localhost:16686.

## Tearing down this example
- `docker-compose down`

## Influences / Credit
The code in this demo was heavily influenced from the example application in https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/examples/demo
which carries the following license.
```
// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Sample contains a simple client that periodically makes a simple http request
// to a server and exports to the OpenTelemetry service.
```
