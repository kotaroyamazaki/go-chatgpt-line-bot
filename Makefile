include .env

deploy:
	gcloud functions deploy chatgpt-bot\
	 --runtime go120 \
	 --trigger-http \
	 --entry-point=Webhook \
	 --memory=256MB \
	 --timeout=180s \
	 --set-env-vars=LINE_CHANNEL_SECRET=${LINE_CHANNEL_SECRET},LINE_CHANNEL_ACCESS_TOKEN=${LINE_CHANNEL_ACCESS_TOKEN},OPENAI_API_KEY=${OPENAI_API_KEY},GCP_PROJECT_ID=${GCP_PROJECT_ID}
