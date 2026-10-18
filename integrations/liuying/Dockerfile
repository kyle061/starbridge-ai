FROM node:22-bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends python3 python3-venv ffmpeg ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY worker/requirements.txt /app/worker/requirements.txt
RUN python3 -m venv /opt/venv && /opt/venv/bin/pip install --no-cache-dir -r /app/worker/requirements.txt
COPY server ./server
COPY worker ./worker
COPY web ./web
RUN mkdir /app/data && chown node:node /app/data
ENV PYTHON_BIN=/opt/venv/bin/python HOST=0.0.0.0 PORT=8787 DATA_DIR=/app/data
USER node
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s CMD node -e "fetch('http://127.0.0.1:8787/api/health').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"
CMD ["node", "server/index.mjs"]
