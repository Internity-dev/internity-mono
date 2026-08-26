const MINIO_URL = import.meta.env.VITE_MINIO_URL ?? 'http://localhost:9000'

// avatars/logos buckets are public-read (see deploy/docker-compose.yml's
// `mc anonymous set download`), so the stored object key resolves straight
// to a public URL — no presigned-URL round trip needed.
export function avatarUrl(key?: string | null): string | undefined {
  return key ? `${MINIO_URL}/internity-avatars/${key}` : undefined
}
