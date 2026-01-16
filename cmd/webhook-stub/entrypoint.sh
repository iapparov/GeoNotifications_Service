#!/bin/bash
# Запускаем webhook и ngrok одновременно

# Запускаем webhook в фоне
./webhook &

# Запускаем ngrok
./ngrok http 9090 --log=stdout