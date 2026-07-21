package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// S3Provider implements the Provider interface for S3-compatible object storage.
// It maps credentials as follows:
// - Email: Access Key ID
// - AccessToken: Secret Access Key
// - RefreshToken: Endpoint URL
// - DisplayName: Bucket Name (saved in SQLite)
type S3Provider struct {
	AccessKey  string
	SecretKey  string
	Endpoint   string
	Bucket     string
	Region     string
}

func NewS3Provider(accessKey, secretKey, endpoint, bucket string) *S3Provider {
	endpoint = strings.TrimSuffix(endpoint, "/")
	return &S3Provider{
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Endpoint:   endpoint,
		Bucket:     bucket,
		Region:     "us-east-1", // standard default region
	}
}

// S3 XML structural parsing definitions
type ListBucketResult struct {
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Contents       []S3Object       `xml:"Contents"`
	CommonPrefixes []S3CommonPrefix `xml:"CommonPrefixes"`
}

type S3Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	Size         int64  `xml:"Size"`
}

type S3CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// signRequest signs an HTTP request using AWS Signature Version 4.
func (s *S3Provider) signRequest(req *http.Request, payloadHash string) error {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	if payloadHash != "" {
		req.Header.Set("x-amz-content-sha256", payloadHash)
	} else {
		req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	}

	parsedURL, err := url.Parse(s.Endpoint)
	if err != nil {
		return err
	}
	req.Header.Set("Host", parsedURL.Host)

	// 1. Create Canonical Request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	// Escape path segments
	escapedPath := ""
	for _, segment := range strings.Split(canonicalURI, "/") {
		if segment != "" {
			escapedPath += "/" + url.PathEscape(segment)
		}
	}
	if strings.HasSuffix(canonicalURI, "/") && !strings.HasSuffix(escapedPath, "/") {
		escapedPath += "/"
	}
	if escapedPath == "" {
		escapedPath = "/"
	}

	canonicalQuery := req.URL.Query().Encode()
	// Replace URL escape '+' with '%20'
	canonicalQuery = strings.ReplaceAll(canonicalQuery, "+", "%20")

	// Sort headers
	headersToSign := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if req.Header.Get("x-amz-copy-source") != "" {
		headersToSign = append(headersToSign, "x-amz-copy-source")
	}
	
	canonicalHeaders := ""
	signedHeaders := ""
	for i, h := range headersToSign {
		val := strings.TrimSpace(req.Header.Get(h))
		canonicalHeaders += fmt.Sprintf("%s:%s\n", h, val)
		if i > 0 {
			signedHeaders += ";"
		}
		signedHeaders += h
	}

	hash := sha256.New()
	if payloadHash != "" {
		hash.Write([]byte(payloadHash))
	} else {
		hash.Write([]byte("UNSIGNED-PAYLOAD"))
	}
	
	canonicalReq := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%x",
		req.Method,
		escapedPath,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		sha256.Sum256([]byte("")), // Use empty string hash for UNSIGNED-PAYLOAD canonical signature
	)

	// 2. Create String to Sign
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%x",
		amzDate,
		credentialScope,
		sha256.Sum256([]byte(canonicalReq)),
	)

	// 3. Calculate Signature
	signingKey := getSignatureKey(s.SecretKey, dateStamp, s.Region, "s3")
	signature := hmacSHA256(signingKey, stringToSign)
	signatureHex := hex.EncodeToString(signature)

	// 4. Set Authorization Header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.AccessKey,
		credentialScope,
		signedHeaders,
		signatureHex,
	)
	req.Header.Set("Authorization", authHeader)

	return nil
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), dateStamp)
	kRegion := hmacSHA256(kDate, regionName)
	kService := hmacSHA256(kRegion, serviceName)
	kSigning := hmacSHA256(kService, "aws4_request")
	return kSigning
}

func (s *S3Provider) doRequest(method, keyPath string, query url.Values, body io.Reader, size int64, headers map[string]string) (*http.Response, error) {
	// Build endpoint path (virtual-host pathing style)
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, s.Bucket, keyPath)
	if strings.HasSuffix(keyPath, "/") && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if size > 0 {
		req.ContentLength = size
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	err = s.signRequest(req, "")
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	return client.Do(req)
}

func (s *S3Provider) GetUserInfo() (name, email string, err error) {
	return "S3 Storage", s.AccessKey, nil
}

func (s *S3Provider) GetQuota() (used, total int64, err error) {
	// S3 Compatible Storages are technically infinite, return a dummy 100 TB limit
	return 0, 100 * 1024 * 1024 * 1024 * 1024, nil
}

func (s *S3Provider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	prefix := remoteParentID
	if prefix == "root" || prefix == "/" {
		prefix = ""
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	query := url.Values{}
	query.Set("list-type", "2")
	query.Set("prefix", prefix)
	query.Set("delimiter", "/")

	resp, err := s.doRequest("GET", "", query, nil, 0, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 list failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var result ListBucketResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []FileMetadata
	// Folders (CommonPrefixes)
	for _, p := range result.CommonPrefixes {
		cleaned := strings.TrimSuffix(p.Prefix, "/")
		name := path.Base(cleaned)
		items = append(items, FileMetadata{
			PhysicalID: p.Prefix,
			Name:       name,
			Size:       0,
			IsFolder:   true,
			ModifiedAt: time.Now(),
		})
	}

	// Files (Contents)
	for _, obj := range result.Contents {
		// List returns the parent prefix itself sometimes. Skip it.
		if obj.Key == prefix {
			continue
		}
		
		name := path.Base(obj.Key)
		// Skip empty folder placeholder objects
		if strings.HasSuffix(obj.Key, "/") && obj.Size == 0 {
			continue
		}

		var modTime time.Time
		if obj.LastModified != "" {
			modTime, _ = time.Parse(time.RFC3339, obj.LastModified)
		}

		items = append(items, FileMetadata{
			PhysicalID: obj.Key,
			Name:       name,
			Size:       obj.Size,
			IsFolder:   false,
			ModifiedAt: modTime,
		})
	}

	return items, nil
}

func (s *S3Provider) CreateFolder(remoteParentID string, name string) (string, error) {
	parent := remoteParentID
	if parent == "root" || parent == "/" {
		parent = ""
	}
	if parent != "" && !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	folderKey := parent + name + "/"

	// In S3, directories are virtual but created by uploading an empty key ending with slash
	resp, err := s.doRequest("PUT", folderKey, nil, nil, 0, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 mkdir failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return folderKey, nil
}

func (s *S3Provider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	parent := remoteParentID
	if parent == "root" || parent == "/" {
		parent = ""
	}
	if parent != "" && !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	fileKey := parent + filename

	resp, err := s.doRequest("PUT", fileKey, nil, r, size, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 upload failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return fileKey, nil
}

func (s *S3Provider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	resp, err := s.doRequest("GET", physicalID, nil, nil, 0, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("s3 download failed: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (s *S3Provider) DeleteFile(physicalID string) error {
	resp, err := s.doRequest("DELETE", physicalID, nil, nil, 0, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *S3Provider) RenameFile(physicalID string, newName string) error {
	parentDir := path.Dir(physicalID)
	if parentDir == "." {
		parentDir = ""
	} else if !strings.HasSuffix(parentDir, "/") {
		parentDir += "/"
	}
	newKey := parentDir + newName

	// 1. Copy object
	copySource := "/" + s.Bucket + "/" + physicalID
	headers := map[string]string{
		"x-amz-copy-source": copySource,
	}
	resp, err := s.doRequest("PUT", newKey, nil, nil, 0, headers)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("s3 copy on rename failed: status %d", resp.StatusCode)
	}

	// 2. Delete original object
	return s.DeleteFile(physicalID)
}

func (s *S3Provider) GetOAuthURL() string {
	return ""
}
