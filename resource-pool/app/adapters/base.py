from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any, Dict

from app.core.generation import GenerationResult, GenerationTaskContext


class BaseAdapter(ABC):
    @abstractmethod
    def generate(self, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:  # noqa: ANN001
        raise NotImplementedError

    def _merge_params(self, ctx: GenerationTaskContext) -> Dict[str, Any]:
        merged: Dict[str, Any] = {}
        merged.update(ctx.default_params or {})
        merged.update(ctx.user_inputs or {})
        merged.update(ctx.admin_fixed or {})
        return merged

    def _resolve_output_filename_prefix(self, ctx: GenerationTaskContext, adapter_name: str) -> str:
        merged = self._merge_params(ctx)
        prefix = merged.get("output_prefix") or merged.get("prefix") or merged.get("scene")
        if prefix is None:
            return adapter_name
        prefix_str = str(prefix).strip()
        return prefix_str or adapter_name


__all__ = ["BaseAdapter", "GenerationResult", "GenerationTaskContext"]

