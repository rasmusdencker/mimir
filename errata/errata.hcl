version = "0.1"

options {
  prefix      = "err-mimir"
  guides_path = "errata/guides/%s.md"
  imports     = [
#    "github.com/grafana/mimir/pkg/mimirpb"
  ]
}

errors "label-name-too-long" {
  message    = "Label \"%.200s\" length exceeds the defined limit in series \"%.200s\""
  cause      = "Received a series whose label name length exceeds the limit"
  categories = ["validation", "label"]
  guide      = file("guides/label-name-too-long.md")
  args       = [
    arg("label", "string"),
    arg("series", "string"),
  ]
  labels     = {
    http_response_code : 400
    level : "warning"
  }
}