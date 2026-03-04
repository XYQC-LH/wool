from __future__ import annotations

from dataclasses import asdict, dataclass, field
from typing import Any, Dict, Optional
from uuid import UUID


@dataclass(frozen=True)
class GenerationTaskContext:
    user_id: UUID
    model_name: str
    default_params: Dict[str, Any] = field(default_factory=dict)
    user_inputs: Dict[str, Any] = field(default_factory=dict)
    admin_fixed: Dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class GenerationResult:
    status: str
    progress: float = 0.0
    message: Optional[str] = None
    result_url: Optional[str] = None
    result_object_key: Optional[str] = None
    result_data: Optional[Dict[str, Any]] = None
    error: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)
