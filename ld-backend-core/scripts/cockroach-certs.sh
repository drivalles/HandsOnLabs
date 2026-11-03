
docker run --rm -v ../e2e/cfg/cockroach/certs:/certs cockroachdb/cockroach:v25.3.7 cert create-ca \
  --certs-dir=/certs \
  --ca-key=/certs/ca.key

docker run --rm -v ../e2e/cfg/cockroach/certs:/certs cockroachdb/cockroach:v25.3.7 cert create-node \
  localhost \
  127.0.0.1 \
  cockroach1 \
  cockroach2 \
  cockroach3 \
  10.5.1.5 \
  10.5.1.6 \
  10.5.1.7 \
  127.90.9.1 \
  127.90.9.2 \
  127.90.9.3 \
  --certs-dir=/certs \
  --ca-key=/certs/ca.key

docker run --rm -v ../e2e/cfg/cockroach/certs:/certs cockroachdb/cockroach:v25.3.7 cert create-client \
  root \
  --certs-dir=/certs \
  --ca-key=/certs/ca.key

sudo chown $USER ../e2e/cfg/cockroach/certs/*