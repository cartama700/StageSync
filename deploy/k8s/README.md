# StageSync K8s Manifests (Phase 14 lite)

Go REST 백엔드를 Kubernetes 에 배포하기 위한 최소 매니페스트 세트. 실 클러스터 없이도 `kubectl --dry-run=client` 로 구조 검증이 가능하도록 구성.

## 파일

| 파일 | 역할 |
|---|---|
| `namespace.yaml` | `stagesync` namespace 생성 (모든 리소스 격리) |
| `configmap.yaml` | 비밀 아닌 런타임 설정 (`LOG_LEVEL`, `LISTEN_ADDR`, `REQUEST_TIMEOUT`, `SHUTDOWN_TIMEOUT`) |
| `deployment.yaml` | `stagesync-server` Deployment (replicas=2, probes, resources, graceful drain 설명 주석 포함) |
| `service.yaml` | ClusterIP `port 80 → http(5050)` 내부 DNS 엔드포인트 |
| `hpa.yaml` | CPU 70% 기준 2 → 10 replicas autoscale |

Secret (`stagesync-secrets`) 은 커밋하지 않고 배포 시 별도 생성.

## 배포 순서

```sh
# 1. namespace
kubectl apply -f deploy/k8s/namespace.yaml

# 2. 시크릿 생성 (MySQL DSN · Redis 주소)
kubectl create secret generic stagesync-secrets \
  --from-literal=mysql-dsn='stagesync:password@tcp(mysql:3306)/stagesync?parseTime=true' \
  --from-literal=redis-addr='redis:6379' \
  -n stagesync

# 3. 나머지 매니페스트 (나머지 4 종은 순서 무관 — kubectl 이 dependency 해결)
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl apply -f deploy/k8s/hpa.yaml
```

혹은 한 줄로: `kubectl apply -f deploy/k8s/` (namespace 포함 전체 적용).

## 드라이런 검증

실 클러스터 없이 YAML 구조만 확인:

```sh
kubectl apply --dry-run=client -f deploy/k8s/
```

client-side 는 서버 스키마를 모르지만 manifest 파싱·필드명·기본 구조 검증은 수행. CI 파이프라인에서 lint 단계로 활용 가능.

## Graceful drain 플로우

Phase 14 lite 의 핵심은 SIGTERM → Shutdown 사이에 readiness gate 로 트래픽을 먼저 비우는 것:

```
SIGTERM 수신
   ↓
readiness.SetDraining()              ← atomic.Bool 이 false 로 전환
   ↓
/health/ready → 503 {"ready": false}  ← kubelet probe 가 실패로 감지 (failureThreshold=2, period=5s)
   ↓
Service endpoint 에서 pod 제거       ← 최대 ~10s 내 LB 전파
   ↓
main 에서 5s sleep                    ← 전파 시간 확보
   ↓
srv.Shutdown(shutdownCtx)             ← in-flight 요청 정리, 최대 SHUTDOWN_TIMEOUT(15s)
   ↓
프로세스 종료
```

총 소요 ≤ `terminationGracePeriodSeconds(60s)` 이내.

### distroless preStop 노트

일반적으론 `lifecycle.preStop.exec.command: ["/bin/sh","-c","sleep 10"]` 로 더 확실하게 대기시키지만, 본 프로젝트 Dockerfile 은 distroless/static 베이스라 `/bin/sh` · `sleep` 바이너리가 없음. 대신 애플리케이션 내부에서 `SetDraining` → `time.Sleep(5s)` → `Shutdown` 순으로 처리해 동일 효과를 달성. 자세한 배경은 `deployment.yaml` 상단 주석 참고.

## 향후 (Phase 14 full 이후)

- Ingress + TLS termination
- PodDisruptionBudget
- 커스텀 메트릭 기반 HPA (RPS / WS 연결 수)
- `DRAIN_DELAY` 를 env 로 승격 (현재 5s 하드코딩)
- NetworkPolicy
