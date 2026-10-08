import logging
from contextlib import asynccontextmanager
from typing import List, Optional

import cv2
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile, status
from pydantic import BaseModel, Field

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("ai-engine")

# Global reference for InsightFace FaceAnalysis model
face_app = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Lifespan context manager to load and warm up InsightFace model on CPU.
    """
    global face_app
    logger.info("Initializing InsightFace FaceAnalysis model: buffalo_l (CPU runtime)...")
    try:
        from insightface.app import FaceAnalysis

        # Initialize with buffalo_l on CPUExecutionProvider
        face_app = FaceAnalysis(name="buffalo_l", providers=["CPUExecutionProvider"])
        face_app.prepare(ctx_id=-1, det_size=(640, 640))
        logger.info("InsightFace buffalo_l model successfully initialized and ready for inference.")
    except Exception as exc:
        logger.error(f"Failed to initialize InsightFace model: {exc}", exc_info=True)
        raise exc

    yield

    logger.info("Shutting down AI Engine microservice...")


app = FastAPI(
    title="FaceAttendance AI Engine",
    description="Microservice for face detection and 512-dim ArcFace embedding extraction using InsightFace buffalo_l",
    version="1.0.0",
    lifespan=lifespan,
)


# --- Request and Response Schemas ---

class ExtractResponse(BaseModel):
    success: bool = True
    detected_faces: int = 1
    embedding: List[float] = Field(..., description="512-dimensional normalized face embedding")


class CompareRequest(BaseModel):
    vector1: List[float] = Field(..., description="First 512-dimensional embedding vector")
    vector2: List[float] = Field(..., description="Second 512-dimensional embedding vector")


class CompareResponse(BaseModel):
    success: bool = True
    similarity: float = Field(..., description="Cosine similarity score (-1.0 to 1.0)")
    is_match: bool = Field(..., description="Whether similarity meets standard threshold (>= 0.65)")


class EnrollMergeResponse(BaseModel):
    success: bool = True
    valid_samples: int = Field(..., description="Number of valid single-face samples processed")
    averaged_embedding: List[float] = Field(..., description="512-dimensional averaged & normalized face embedding")


# --- Helper Utilities ---

async def decode_upload_image(file: UploadFile) -> np.ndarray:
    """
    Read uploaded file bytes and decode into OpenCV BGR numpy array.
    """
    contents = await file.read()
    if not contents:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Uploaded file '{file.filename}' is empty."
        )

    nparr = np.frombuffer(contents, np.uint8)
    image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

    if image is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Failed to decode image from '{file.filename}'. Ensure the file is a valid image format (JPEG, PNG, etc.)."
        )

    return image


def extract_embedding_from_image(image: np.ndarray, file_label: str = "image") -> List[float]:
    """
    Detects faces in the image, strictly checks for exactly 1 face, and returns L2-normalized 512-d embedding.
    """
    if face_app is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="AI Engine face model is not loaded."
        )

    faces = face_app.get(image)
    num_faces = len(faces)

    if num_faces == 0:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"No face detected in {file_label}. Please ensure the face is clearly visible, well-lit, and facing the camera."
        )

    if num_faces > 1:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Multiple faces detected ({num_faces} faces) in {file_label}. Exactly one face must be present."
        )

    face = faces[0]
    embedding = face.embedding

    if embedding is None or len(embedding) != 512:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Face embedding extraction failed or dimension is not 512."
        )

    # Normalize vector to unit sphere (L2 norm)
    norm = np.linalg.norm(embedding)
    if norm > 0:
        embedding = embedding / norm

    return embedding.astype(float).tolist()


# --- Endpoints ---

@app.get("/health")
async def health_check():
    """
    Health check endpoint for container probes and orchestration.
    """
    return {
        "status": "ok",
        "service": "ai-engine",
        "model_loaded": face_app is not None,
    }


@app.post("/extract", response_model=ExtractResponse)
async def extract_face(
    image: Optional[UploadFile] = File(None),
    file: Optional[UploadFile] = File(None),
):
    """
    Extract a 512-dimensional face embedding from an uploaded image.
    Strictly verifies that exactly one face is detected.
    Accepts multipart form-data under field name 'image' or 'file'.
    """
    upload = image or file
    if upload is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Image file is required. Please upload via form field 'image' or 'file'."
        )

    img = await decode_upload_image(upload)
    embedding = extract_embedding_from_image(img, file_label=f"'{upload.filename or 'image'}'")

    return ExtractResponse(
        success=True,
        detected_faces=1,
        embedding=embedding,
    )


@app.post("/compare", response_model=CompareResponse)
async def compare_embeddings(request: CompareRequest):
    """
    Compute cosine similarity between two 512-dimensional face embedding vectors.
    Returns cosine similarity score (-1.0 to 1.0) and whether it matches the 0.65 threshold.
    """
    if len(request.vector1) != 512:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"vector1 must have exactly 512 elements, received {len(request.vector1)}."
        )

    if len(request.vector2) != 512:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"vector2 must have exactly 512 elements, received {len(request.vector2)}."
        )

    v1 = np.array(request.vector1, dtype=np.float32)
    v2 = np.array(request.vector2, dtype=np.float32)

    norm1 = np.linalg.norm(v1)
    norm2 = np.linalg.norm(v2)

    if norm1 == 0.0 or norm2 == 0.0:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid vector: norm is zero."
        )

    # Cosine similarity formula: dot(v1, v2) / (norm(v1) * norm(v2))
    similarity = float(np.dot(v1, v2) / (norm1 * norm2))
    similarity = float(np.clip(similarity, -1.0, 1.0))

    # Standard face verification threshold from SPEC.md (0.65)
    is_match = similarity >= 0.65

    return CompareResponse(
        success=True,
        similarity=round(similarity, 4),
        is_match=is_match,
    )


@app.post("/enroll-merge", response_model=EnrollMergeResponse)
async def enroll_merge(images: List[UploadFile] = File(...)):
    """
    Enrollment helper endpoint according to SPEC.md section 4.1.
    Accepts 3-5 face photos, extracts embeddings for single-face photos,
    and returns the averaged and re-normalized 512-d embedding.
    """
    if not images or len(images) == 0:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="At least one image is required for enrollment merge."
        )

    embeddings = []
    errors = []

    for idx, upload in enumerate(images):
        try:
            img = await decode_upload_image(upload)
            emb = extract_embedding_from_image(img, file_label=f"image #{idx + 1} ('{upload.filename}')")
            embeddings.append(np.array(emb, dtype=np.float32))
        except HTTPException as err:
            errors.append(f"Image {idx + 1}: {err.detail}")

    if not embeddings:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail={
                "message": "No valid face samples could be extracted from the uploaded images.",
                "errors": errors,
            }
        )

    # Calculate average embedding across all valid samples
    avg_vector = np.mean(embeddings, axis=0)
    norm = np.linalg.norm(avg_vector)
    if norm > 0:
        avg_vector = avg_vector / norm

    return EnrollMergeResponse(
        success=True,
        valid_samples=len(embeddings),
        averaged_embedding=avg_vector.astype(float).tolist(),
    )
