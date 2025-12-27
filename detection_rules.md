# Detection Rules Examples

## 1. Authentication & Access Control

### Rule: Multiple Failed Login Attempts

```yaml
rule:
  id: auth-001
  name: "Brute Force Login Detection"
  description: "Detects multiple failed login attempts from the same source"
  category: authentication
  severity: high
  
  conditions:
    - field: event_type
      operator: equals
      value: "authentication_failed"
    - field: source_ip
      operator: count_distinct
      count_field: username
      threshold: 5
      timewindow: 5m
  
  actions:
    - type: create_incident
      priority: high
    - type: execute_playbook
      playbook: "brute-force-response"
    - type: notify
      channels: ["slack", "email"]
  
  metadata:
    mitre_attack: T1110
    references:
      - "https://attack.mitre.org/techniques/T1110/"
```

### Rule: Unusual Login Time

```yaml
rule:
  id: auth-002
  name: "Login Outside Business Hours"
  description: "Detects successful logins outside normal business hours"
  category: authentication
  severity: medium
  
  conditions:
    - field: event_type
      operator: equals
      value: "authentication_success"
    - field: timestamp
      operator: time_range
      outside_of:
        start: "08:00"
        end: "18:00"
        timezone: "UTC"
        days: ["monday", "tuesday", "wednesday", "thursday", "friday"]
    - field: user.risk_score
      operator: greater_than
      value: 50
  
  actions:
    - type: enrich_user_context
    - type: create_alert
      priority: medium
    - type: notify
      channels: ["security_team"]
```

### Rule: Privileged Account Usage

```yaml
rule:
  id: auth-003
  name: "Privileged Account Activity"
  description: "Monitors privileged account usage for suspicious activity"
  category: privileged_access
  severity: high
  
  conditions:
    - field: user.role
      operator: in
      values: ["admin", "root", "domain_admin"]
    - field: action
      operator: in
      values: ["user_creation", "permission_change", "group_modification"]
  
  actions:
    - type: create_incident
      priority: high
    - type: require_approval
      approvers: ["security_team"]
    - type: log_activity
```

## 2. Network Security

### Rule: Port Scan Detection

```yaml
rule:
  id: net-001
  name: "Port Scanning Activity"
  description: "Detects port scanning behavior"
  category: reconnaissance
  severity: high
  
  conditions:
    - field: event_type
      operator: equals
      value: "network_connection"
    - field: source_ip
      operator: count_distinct
      count_field: destination_port
      threshold: 20
      timewindow: 1m
  
  actions:
    - type: create_incident
      priority: high
    - type: execute_playbook
      playbook: "network-isolation"
    - type: block_ip
      duration: "1h"
```

### Rule: Data Exfiltration

```yaml
rule:
  id: net-002
  name: "Unusual Outbound Data Transfer"
  description: "Detects large volumes of outbound data transfer"
  category: exfiltration
  severity: critical
  
  conditions:
    - field: direction
      operator: equals
      value: "outbound"
    - field: bytes_transferred
      operator: sum
      threshold: 10737418240  # 10GB
      timewindow: 10m
      group_by: source_ip
    - field: destination
      operator: not_in
      values: ["approved_backup_servers", "cdn_endpoints"]
  
  actions:
    - type: create_incident
      priority: critical
    - type: execute_playbook
      playbook: "data-exfiltration-response"
    - type: network_throttle
      bandwidth_limit: "100Mbps"
```

### Rule: Connection to Known Malicious IP

```yaml
rule:
  id: net-003
  name: "Malicious IP Communication"
  description: "Detects connections to known malicious IPs"
  category: command_and_control
  severity: critical
  
  conditions:
    - field: event_type
      operator: equals
      value: "network_connection"
    - field: destination_ip
      operator: in_threat_intel
      sources: ["alienvault", "abuse.ch", "emergingthreats"]
  
  actions:
    - type: create_incident
      priority: critical
    - type: block_connection
    - type: isolate_host
    - type: execute_playbook
      playbook: "malware-response"
```

## 3. Malware & Endpoint Security

### Rule: Suspicious Process Execution

```yaml
rule:
  id: mal-001
  name: "Suspicious Process Behavior"
  description: "Detects processes with suspicious characteristics"
  category: malware
  severity: high
  
  conditions:
    - field: process.name
      operator: regex
      pattern: "^[a-f0-9]{8,}\\.exe$"  # Random hex filename
    - field: process.parent
      operator: in
      values: ["cmd.exe", "powershell.exe", "wscript.exe"]
    - field: process.command_line
      operator: contains_any
      values: ["-enc", "-encodedcommand", "downloadstring", "iex"]
  
  actions:
    - type: create_incident
      priority: high
    - type: kill_process
    - type: quarantine_file
    - type: execute_playbook
      playbook: "malware-containment"
```

### Rule: Ransomware Indicators

```yaml
rule:
  id: mal-002
  name: "Ransomware Activity Detection"
  description: "Detects potential ransomware behavior"
  category: ransomware
  severity: critical
  
  conditions:
    - field: file.operation
      operator: equals
      value: "modification"
    - field: file.extension
      operator: in
      values: [".encrypted", ".locked", ".crypto", ".crypt"]
    - field: file.count
      operator: greater_than
      threshold: 10
      timewindow: 1m
      group_by: host
  
  actions:
    - type: create_incident
      priority: critical
    - type: isolate_host
    - type: disable_network
    - type: snapshot_system
    - type: execute_playbook
      playbook: "ransomware-response"
    - type: page_oncall
```

## 4. Email Security

### Rule: Phishing Email Detection

```yaml
rule:
  id: email-001
  name: "Potential Phishing Email"
  description: "Detects emails with phishing indicators"
  category: phishing
  severity: medium
  
  conditions:
    - field: email.subject
      operator: contains_any
      values: ["urgent action required", "verify your account", "suspended"]
    - field: email.sender.domain
      operator: similarity
      compare_to: "known_legitimate_domains"
      threshold: 0.85  # Typosquatting detection
    - field: email.attachments
      operator: exists
    - field: email.attachments[].extension
      operator: in
      values: [".exe", ".scr", ".zip", ".iso"]
  
  actions:
    - type: quarantine_email
    - type: create_alert
      priority: medium
    - type: notify
      recipients: ["security_team"]
```

### Rule: Business Email Compromise

```yaml
rule:
  id: email-002
  name: "BEC Detection"
  description: "Detects potential business email compromise"
  category: bec
  severity: critical
  
  conditions:
    - field: email.sender.display_name
      operator: matches
      value: "executive_names"
    - field: email.sender.address
      operator: not_in
      values: ["authorized_executive_emails"]
    - field: email.body
      operator: contains_any
      values: ["wire transfer", "urgent payment", "invoice attached"]
  
  actions:
    - type: quarantine_email
    - type: create_incident
      priority: critical
    - type: notify
      recipients: ["cfo", "security_team"]
      method: "phone_call"
```

## 5. Cloud Security

### Rule: Unusual AWS API Activity

```yaml
rule:
  id: cloud-001
  name: "AWS Privilege Escalation"
  description: "Detects potential AWS privilege escalation"
  category: privilege_escalation
  severity: high
  
  conditions:
    - field: cloud.provider
      operator: equals
      value: "aws"
    - field: cloud.event_name
      operator: in
      values:
        - "AttachUserPolicy"
        - "CreateAccessKey"
        - "CreateLoginProfile"
        - "UpdateAssumeRolePolicy"
    - field: cloud.user
      operator: not_in
      values: ["authorized_admin_users"]
  
  actions:
    - type: create_incident
      priority: high
    - type: revoke_credentials
    - type: execute_playbook
      playbook: "aws-incident-response"
```

### Rule: Public S3 Bucket Exposure

```yaml
rule:
  id: cloud-002
  name: "S3 Bucket Made Public"
  description: "Detects S3 buckets being made publicly accessible"
  category: data_exposure
  severity: critical
  
  conditions:
    - field: cloud.event_name
      operator: in
      values: ["PutBucketAcl", "PutBucketPolicy"]
    - field: cloud.request_parameters
      operator: contains
      value: "AllUsers"
  
  actions:
    - type: create_incident
      priority: critical
    - type: revert_change
    - type: notify
      recipients: ["security_team", "data_owner"]
```

## 6. Insider Threat

### Rule: Mass Data Download

```yaml
rule:
  id: insider-001
  name: "Bulk Data Download"
  description: "Detects unusual bulk data downloads"
  category: insider_threat
  severity: high
  
  conditions:
    - field: action
      operator: equals
      value: "file_download"
    - field: file.count
      operator: greater_than
      threshold: 100
      timewindow: 1h
      group_by: user
    - field: file.total_size
      operator: greater_than
      threshold: 1073741824  # 1GB
  
  actions:
    - type: create_incident
      priority: high
    - type: notify
      recipients: ["manager", "hr", "security_team"]
    - type: log_detailed_activity
```

### Rule: After-Hours Sensitive Data Access

```yaml
rule:
  id: insider-002
  name: "Off-Hours Sensitive Access"
  description: "Access to sensitive data outside business hours"
  category: insider_threat
  severity: medium
  
  conditions:
    - field: data.classification
      operator: in
      values: ["confidential", "restricted", "top_secret"]
    - field: timestamp
      operator: outside_hours
      business_hours:
        start: "08:00"
        end: "18:00"
    - field: user.recent_termination
      operator: equals
      value: true
  
  actions:
    - type: create_alert
      priority: high
    - type: require_justification
    - type: notify
      recipients: ["security_team"]
```

## 7. Application Security

### Rule: SQL Injection Attempt

```yaml
rule:
  id: app-001
  name: "SQL Injection Detection"
  description: "Detects potential SQL injection attempts"
  category: application_attack
  severity: high
  
  conditions:
    - field: http.request.uri
      operator: regex
      pattern: "('.*(--|\\/\\*|;|\\bunion\\b|\\bselect\\b|\\binsert\\b|\\bupdate\\b|\\bdelete\\b).*)"
    - field: http.response.status
      operator: equals
      value: 500
  
  actions:
    - type: create_incident
      priority: high
    - type: block_ip
      duration: "24h"
    - type: waf_rule_update
```

### Rule: Credential Stuffing

```yaml
rule:
  id: app-002
  name: "Credential Stuffing Attack"
  description: "Detects credential stuffing attempts"
  category: credential_access
  severity: high
  
  conditions:
    - field: event_type
      operator: equals
      value: "login_attempt"
    - field: source_ip
      operator: count_distinct
      count_field: username
      threshold: 50
      timewindow: 5m
    - field: user_agent
      operator: in
      values: ["curl", "python-requests", "automated_tool_signatures"]
  
  actions:
    - type: create_incident
      priority: high
    - type: rate_limit
      limit: "10/minute"
    - type: captcha_challenge
```

## 8. Compliance & Policy

### Rule: PCI DSS Violation

```yaml
rule:
  id: compliance-001
  name: "PCI DSS Access Violation"
  description: "Detects access to cardholder data environment violations"
  category: compliance
  severity: critical
  
  conditions:
    - field: resource.tag
      operator: contains
      value: "pci-cde"
    - field: user.pci_training
      operator: equals
      value: false
  
  actions:
    - type: block_access
    - type: create_incident
      priority: critical
    - type: notify
      recipients: ["compliance_team", "ciso"]
```

### Rule: Data Retention Policy Violation

```yaml
rule:
  id: compliance-002
  name: "Data Retention Violation"
  description: "Detects data retention policy violations"
  category: compliance
  severity: medium
  
  conditions:
    - field: file.age
      operator: greater_than
      value: 2555  # 7 years in days
    - field: file.classification
      operator: equals
      value: "financial_record"
    - field: file.location
      operator: not_in
      values: ["archive_storage"]
  
  actions:
    - type: create_alert
      priority: medium
    - type: trigger_archival
    - type: notify
      recipients: ["compliance_team"]
```

## Detection Rule Best Practices

### 1. Rule Design

- Keep conditions specific but not overly restrictive
- Use appropriate time windows
- Consider false positive rates
- Test rules in monitoring mode first

### 2. Severity Calibration

- **Critical**: Immediate security threat, requires instant response
- **High**: Significant security concern, response within 15 minutes
- **Medium**: Potential security issue, response within 1 hour
- **Low**: Anomaly requiring investigation, response within 24 hours

### 3. Action Selection

- Start with monitoring and alerting
- Gradually introduce automated responses
- Always have rollback mechanisms
- Require approval for critical actions

### 4. Maintenance

- Review rules quarterly
- Update threat intelligence feeds
- Tune thresholds based on false positives
- Archive deprecated rules
