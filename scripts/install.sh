#!/bin/bash

go build -o bin/venv-manager cmd/venv-manager/main.go
sudo mv bin/venv-manager /usr/local/bin/

# TODO: how can i manage better this functionality?
#sudo cat > /usr/local/bin/v-manager << 'EOF'
##!/bin/bash
#if [ "$1" = "activate" ]; then
#    eval "$(venv-manager activate $2)"
#else
#    venv-manager "$@"
#fi
#EOF
#chmod +x /usr/local/bin/v-manager