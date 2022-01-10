
Build app
```
GOOS=linux CGO_ENABLED=0 go build cloudwatch-s3-exporter.go
```

Compress and upload zip archive to s3 bucket for deploy:
```
zip cloudwatch-s3-exporter.zip cloudwatch-s3-exporter
```