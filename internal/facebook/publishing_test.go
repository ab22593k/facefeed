package facebook

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"facefeed/internal/testutil"
)

// ---------------------------------------------------------------------------
// PostVideoUpload tests
// ---------------------------------------------------------------------------

func TestPostVideoUpload_Success(t *testing.T) {
	imgPath := createTempFile(t, "video.mp4", "fake-mp4-data")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/videos")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "vid_123"}`))
	})
	defer server.Close()

	id, err := client.PostVideoUpload("page_1", "My Video", "A great video", imgPath, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "vid_123" {
		t.Errorf("got id %q, want %q", id, "vid_123")
	}
}

func TestPostVideoUpload_WithSchedule(t *testing.T) {
	imgPath := createTempFile(t, "sched_vid.mp4", "scheduled-video")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "vid_sched_1"}`))
	})
	defer server.Close()

	id, err := client.PostVideoUpload("page_1", "Scheduled", "desc", imgPath, 2000000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "vid_sched_1" {
		t.Errorf("got id %q, want %q", id, "vid_sched_1")
	}
}

func TestPostVideoUpload_APIError(t *testing.T) {
	imgPath := createTempFile(t, "error_vid.mp4", "will-fail")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid video", "code": 100},
		})
	})
	defer server.Close()

	_, err := client.PostVideoUpload("page_1", "Bad", "desc", imgPath, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected status 400 in error, got: %v", err)
	}
}

func TestPostVideoUpload_FileNotFound(t *testing.T) {
	client := New(&http.Client{Timeout: time.Second}, "tok")
	_, err := client.PostVideoUpload("page_1", "Test", "desc", "/tmp/nonexistent_video_xyz.mp4", 0)
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}

func TestPostVideoUpload_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "netfail_vid.mp4", "will-timeout")

	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.PostVideoUpload("page_1", "Test", "desc", imgPath, 0)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PublishReel tests
// ---------------------------------------------------------------------------

func TestPublishReel_Success(t *testing.T) {
	imgPath := createTempFile(t, "reel.mp4", "fake-reel-data")

	callCount := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		phase := r.FormValue("upload_phase")
		switch phase {
		case "start":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"video_id":   "reel_123",
				"upload_url": "https://rupload.facebook.com/reel_123",
			})
		case "finish":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		default:
			// Phase 2: file upload
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}
	})
	defer server.Close()

	id, err := client.PublishReel("page_1", "My Reel", imgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "reel_123" {
		t.Errorf("got id %q, want %q", id, "reel_123")
	}
}

func TestPublishReel_StartError(t *testing.T) {
	imgPath := createTempFile(t, "reel_fail.mp4", "fail")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid reel", "code": 100}}`))
	})
	defer server.Close()

	_, err := client.PublishReel("page_1", "Fail", imgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPublishReel_FinishError(t *testing.T) {
	imgPath := createTempFile(t, "reel_finish_fail.mp4", "fail")

	callCount := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		phase := r.FormValue("upload_phase")
		if callCount == 1 {
			// Phase 1: start - return video_id and upload_url
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"video_id":   "reel_123",
				"upload_url": "https://rupload.facebook.com/reel_123",
			})
		} else if callCount == 2 || phase == "" {
			// Phase 2: file upload
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		} else {
			// Phase 3: finish fails
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "Finish failed"}}`))
		}
	})
	defer server.Close()

	_, err := client.PublishReel("page_1", "Fail", imgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPublishReel_FileNotFound(t *testing.T) {
	client := New(&http.Client{Timeout: time.Second}, "tok")
	_, err := client.PublishReel("page_1", "Test", "/tmp/nonexistent_reel_xyz.mp4")
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}

func TestPublishReel_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "reel_net.mp4", "timeout")
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.PublishReel("page_1", "Test", imgPath)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PublishStory tests
// ---------------------------------------------------------------------------

func TestPublishStory_Video_Success(t *testing.T) {
	imgPath := createTempFile(t, "story.mp4", "fake-story-video")

	callCount := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		phase := r.FormValue("upload_phase")
		switch phase {
		case "start":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"video_id":   "story_123",
				"upload_url": "https://rupload.facebook.com/story_123",
			})
		case "finish":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true, "post_id": "story_post_1"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}
	})
	defer server.Close()

	id, err := client.PublishStory("page_1", imgPath, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "story_post_1" {
		t.Errorf("got id %q, want %q", id, "story_post_1")
	}
}

func TestPublishStory_Photo_Success(t *testing.T) {
	imgPath := createTempFile(t, "story.jpg", "fake-story-photo")

	callCount := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "/photos") {
			// Step 1: Upload draft photo (published=false via multipart)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "photo_draft_1"}`))
		} else if strings.Contains(r.URL.Path, "/photo_stories") {
			// Step 2: Create story from draft
			testutil.AssertFormValue(t, r, "photo_id", "photo_draft_1")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "story_photo_1"}`))
		}
	})
	defer server.Close()

	id, err := client.PublishStory("page_1", imgPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "story_photo_1" {
		t.Errorf("got id %q, want %q", id, "story_photo_1")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestPublishStory_APIError(t *testing.T) {
	imgPath := createTempFile(t, "story_fail.mp4", "fail")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid story"}}`))
	})
	defer server.Close()

	_, err := client.PublishStory("page_1", imgPath, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ReplyToComment tests
// ---------------------------------------------------------------------------

func TestReplyToComment_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/comments")
		testutil.AssertFormValue(t, r, "message", "Thanks for the feedback!")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "reply_123"}`))
	})
	defer server.Close()

	id, err := client.ReplyToComment("comment_456", "Thanks for the feedback!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "reply_123" {
		t.Errorf("got id %q, want %q", id, "reply_123")
	}
}

func TestReplyToComment_APIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"message": "No permission to reply", "code": 200}}`))
	})
	defer server.Close()

	_, err := client.ReplyToComment("comment_456", "Reply")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in error, got: %v", err)
	}
}

func TestReplyToComment_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.ReplyToComment("comment_456", "Reply")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdatePost tests
// ---------------------------------------------------------------------------

func TestUpdatePost_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertFormValue(t, r, "message", "Updated message")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})
	defer server.Close()

	err := client.UpdatePost("post_123", "Updated message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdatePost_APIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Post cannot be edited", "code": 100},
		})
	})
	defer server.Close()

	err := client.UpdatePost("post_123", "New message")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected 400 in error, got: %v", err)
	}
}

func TestUpdatePost_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	err := client.UpdatePost("post_123", "New message")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetInsights tests
// ---------------------------------------------------------------------------

func TestGetInsights_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodGet)
		if !strings.Contains(r.URL.RawQuery, "page_impressions") {
			t.Errorf("expected page_impressions in query, got: %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "page_fan_adds") {
			t.Errorf("expected page_fan_adds in query, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"name": "page_impressions",
					"period": "day",
					"values": [{"value": 1500, "end_time": "2026-06-05T07:00:00+0000"}]
				},
				{
					"name": "page_fan_adds",
					"period": "day",
					"values": [{"value": 25, "end_time": "2026-06-05T07:00:00+0000"}]
				}
			]
		}`))
	})
	defer server.Close()

	insights, err := client.GetInsights("page_123", "page_impressions,page_fan_adds", "day")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insights) != 2 {
		t.Fatalf("got %d insights, want 2", len(insights))
	}
	if insights[0].Name != "page_impressions" {
		t.Errorf("insights[0].Name = %q, want %q", insights[0].Name, "page_impressions")
	}
	if insights[1].Name != "page_fan_adds" {
		t.Errorf("insights[1].Name = %q, want %q", insights[1].Name, "page_fan_adds")
	}
	if len(insights[0].Values) != 1 {
		t.Errorf("expected 1 value for page_impressions, got %d", len(insights[0].Values))
	}
}

func TestGetInsights_EmptyResponse(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	})
	defer server.Close()

	insights, err := client.GetInsights("page_123", "page_impressions", "day")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insights) != 0 {
		t.Errorf("got %d insights, want 0", len(insights))
	}
}

func TestGetInsights_APIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid metric", "code": 100}}`))
	})
	defer server.Close()

	_, err := client.GetInsights("page_123", "invalid_metric", "day")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid metric") {
		t.Errorf("expected 'Invalid metric' in error, got: %v", err)
	}
}

func TestGetInsights_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.GetInsights("page_123", "page_impressions", "day")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetInsights_InvalidJSON(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{garbage}`))
	})
	defer server.Close()

	_, err := client.GetInsights("page_123", "page_impressions", "day")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
