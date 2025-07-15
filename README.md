```bash
docker build -t learn-go-api:latest .
```

```bash
docker run -d -p 8080:8080 --name learn-go-api learn-go-api:latest
```