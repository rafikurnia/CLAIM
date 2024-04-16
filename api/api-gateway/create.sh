#!/bin/bash

gcloud api-gateway apis create measurement-platform --project=omega-moonlight-291117

gcloud api-gateway apis describe measurement-platform --project=omega-moonlight-291117

gcloud api-gateway api-configs create measurement-platform \
  --api=measurement-platform --openapi-spec=openapi2-measurement.yaml \
  --project=omega-moonlight-291117 --backend-auth-service-account=deployer@omega-moonlight-291117.iam.gserviceaccount.com

gcloud api-gateway api-configs describe measurement-platform \
  --api=measurement-platform --project=omega-moonlight-291117

gcloud api-gateway gateways create measurement-platform \
  --api=measurement-platform --api-config=measurement-platform \
  --location=europe-west1 --project=omega-moonlight-291117

gcloud api-gateway gateways describe measurement-platform \
  --location=europe-west1 --project=omega-moonlight-291117
