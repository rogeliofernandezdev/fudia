# Red

Solo frontends y API reciben tráfico externo. Base, colas, caché y observabilidad
usan red privada en la arquitectura objetivo. Excepción MVP GCP: Cloud SQL con
endpoint público, accesible desde Cloud Run mediante Auth Proxy administrado y
socket, sin VPC. No abrir `0.0.0.0/0`; DBeaver directo usa IP del operador y TLS.
TLS obligatorio en la red remota y CORS limitado a orígenes conocidos.
