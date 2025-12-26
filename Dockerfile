FROM squidfunk/mkdocs-material:latest

COPY docs/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt
