# Tekton buildcop slack bot

This is the Slack buildcop bot code and configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: buildcop-cfg
  namespace: default
data:
  SLACKTOKEN: …token…
  BOTID: URCPZNB37
  CHANNELID: CPY3T4YHM
```
