{
  "subject": {"commonName": {{ toJson .Token.sparkly.device_id }}},
  "sans": [{"type": "uri", "value": {{ toJson .Token.sparkly.uri }}}],
  "keyUsage": ["digitalSignature"],
  "extKeyUsage": ["clientAuth"],
  "basicConstraints": {"isCA": false}
}
