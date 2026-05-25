#!/usr/bin/env nu

open .env
| lines
| where {|l| not ($l | str starts-with "#") and ($l | is-not-empty) }
| each { split row -n 2 "=" }
| each { {($in.0 | str trim): ($in.1 | str trim | str trim --char '"')} }
| reduce --fold {} {|row acc| $acc | merge $row }
| load-env

let body = {
  model: $env.EDGE_AI_MODEL
  messages: [
    {
      role: "system"
      content: "Return JSON only. Output exactly
 {\"signals\":[{\"index\":0,\"thesis\":\"x\",\"why_it_matters\":\"y\",\"match_type\":\"watchlist\",\"score_boost\":0}]}"
    }
    {
      role: "user"
      content: "Candidates:\n[{\"index\":0,\"headline\":\"Bitcoin rises after peace agreement\"}]"
    }
  ]
  response_format: {
    type: "json_object"
    json_schema: {
      name: "signal_judgment"
      schema: {
        type: "object"
        properties: {
          signals: {
            type: "array"
            items: {
              type: "object"
              properties: {
                index: {type: "integer"}
                thesis: {type: "string"}
                why_it_matters: {type: "string"}
                match_type: {type: "string" enum: ["market-linked" "watchlist" "no-match"]}
                score_boost: {type: "number"}
              }
              required: ["index" "thesis" "why_it_matters" "match_type" "score_boost"]
            }
          }
        }
        required: ["signals"]
      }
    }
  }
} | to json

(
  curl
  --silent
  --show-error
  --location
  --request POST
  --url $"($env.EDGE_AI_BASE_URL)/chat/completions"
  --header $"Authorization: Bearer ($env.EDGE_AI_API_KEY)"
  --header "Content-Type: application/json"
  --data $body
)
