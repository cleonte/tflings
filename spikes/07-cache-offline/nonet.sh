mkdir -p /w && cd /w && cat > main.tf <<'TF'
resource "terraform_data" "x" { input = "hi" }
output "o" { value = terraform_data.x.input }
TF
env -i PATH=/usr/bin:/bin HOME=/tmp CHECKPOINT_DISABLE=1 TF_DATA_DIR=/w/.d $T init -no-color -input=false 2>&1 | grep -E 'built in|Error|successfully'
env -i PATH=/usr/bin:/bin HOME=/tmp CHECKPOINT_DISABLE=1 TF_DATA_DIR=/w/.d $T plan -no-color -input=false 2>&1 | grep -E 'Plan:|Error'
