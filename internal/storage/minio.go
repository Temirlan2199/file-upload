// Package storage — хранение самих файлов (байтов) в MinIO.
// MinIO говорит на протоколе Amazon S3, поэтому этот же код
// работал бы и с настоящим AWS S3 — поменялись бы только адрес и ключи.
package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client *minio.Client
	bucket string // bucket — «корзина», что-то вроде папки верхнего уровня в S3
}

// NewMinioStorage подключается к MinIO и создаёт bucket, если его ещё нет.
func NewMinioStorage(ctx context.Context, endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL, // false — http, true — https
	})
	if err != nil {
		return nil, err
	}

	// Проверяем, есть ли bucket. Это заодно проверяет, что MinIO доступен.
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &MinioStorage{client: client, bucket: bucket}, nil
}

// Upload сохраняет данные из r под ключом key.
// Файл не читается целиком в память — MinIO-клиент читает r кусками и отправляет по сети.
func (s *MinioStorage) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Download возвращает поток для чтения объекта.
// Вызывающий обязан закрыть его (Close), иначе соединение с MinIO «утечёт».
func (s *MinioStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *MinioStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
