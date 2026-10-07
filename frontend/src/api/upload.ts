import { api, call } from './client'

export type UploadPurpose = 'avatar' | 'course_cover' | 'material'

export type Uploaded = {
  /** the key to save with the profile, course or material */
  objectKey: string
  fileName: string
  size: number
  contentType: string
}

/**
 * Uploads a file in two steps: the API gives an address, the file goes straight
 * to the file storage. The returned key is sent when the profile, the course or
 * the lesson is saved.
 */
export async function uploadFile(purpose: UploadPurpose, file: File): Promise<Uploaded> {
  const contentType = file.type || 'application/octet-stream'

  const presign = await call(
    api.POST('/uploads/presign', { body: { purpose, file_name: file.name, content_type: contentType } }),
  )

  if (presign.max_file_size && file.size > presign.max_file_size) {
    throw new Error(`The file is too large. The most is ${Math.round(presign.max_file_size / 1024 / 1024)} MB.`)
  }

  const response = await fetch(presign.upload_url!, { method: 'PUT', headers: { 'Content-Type': contentType }, body: file })

  if (!response.ok) throw new Error('The file could not be uploaded. Try again.')

  return { objectKey: presign.object_key!, fileName: file.name, size: file.size, contentType }
}
