from __future__ import annotations

from typing import Any, Dict, Optional
from uuid import UUID

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from app.adapters.banana import BananaAdapter
from app.adapters.jimeng import JimengAdapter
from app.adapters.seedream import SeedreamAdapter
from app.adapters.sora2 import Sora2Adapter
from app.core.generation import GenerationTaskContext
from app.jimeng_api import router as jimeng_router

app = FastAPI(title="Wool Resource Pool", version="0.1.0")
app.include_router(jimeng_router)


class GenerateRequest(BaseModel):
    provider: str = Field(..., description="banana | seedream | sora2 | jimeng")
    user_id: UUID
    model_name: str = Field("", description="适配器的模型名（如 seedream-4 / nano-banana / jimeng-4.0 / ...）")
    default_params: Dict[str, Any] = Field(default_factory=dict)
    user_inputs: Dict[str, Any] = Field(default_factory=dict)
    admin_fixed: Dict[str, Any] = Field(default_factory=dict)
    adapter_settings: Optional[Dict[str, Any]] = Field(default=None, description="可选：覆盖 adapter 配置（仅用于调试）")


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok"}


@app.post("/v1/generate")
def generate(req: GenerateRequest) -> Dict[str, Any]:
    provider = (req.provider or "").strip().lower()
    adapter_cls = {
        "banana": BananaAdapter,
        "jimeng": JimengAdapter,
        "seedream": SeedreamAdapter,
        "sora2": Sora2Adapter,
    }.get(provider)

    if not adapter_cls:
        raise HTTPException(status_code=400, detail=f"unsupported provider: {req.provider}")

    ctx = GenerationTaskContext(
        user_id=req.user_id,
        model_name=req.model_name,
        default_params=req.default_params,
        user_inputs=req.user_inputs,
        admin_fixed=req.admin_fixed,
    )

    adapter = adapter_cls(settings=req.adapter_settings)
    result = adapter.generate(ctx)
    return {"data": result.to_dict()}
