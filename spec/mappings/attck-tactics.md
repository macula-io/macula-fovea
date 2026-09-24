# Mapping: ATT&CK tactics

ATT&CK's tactics (Reconnaissance, Resource Development, Initial Access,
Execution, Persistence, Privilege Escalation, Defense Evasion, Credential
Access, Discovery, Lateral Movement, Collection, Command and Control,
Exfiltration, Impact) sit *below* fovea's grid: fovea asks "what is this
cell's threat in the general sense," ATT&CK answers "which *techniques* from
the same class have actually been observed". Link from a cell's
manifestations to ATT&CK technique IDs when:

- the manifestation is a *recognized technique* (e.g. DDoS → ATT&CK has an ID),
- you want incident-response teams to find the cell from the technique,
- you need concrete detection artifacts that ATT&CK's data-source tables
  reference.

No formal axis-to-axis mapping is claimed: ATT&CK is technique-space and
observation-space; fovea is threat-model space. The one rule: a cell that
links to a *technique* must still have its own manifestations, properly
named — "map not duplicate" holds here too.
