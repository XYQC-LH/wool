from __future__ import annotations

import copy
from dataclasses import asdict
from functools import lru_cache
from typing import Any, Dict, Optional

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from app.adapters.jimeng import (  # noqa: PLC2701
    JimengAdapter,
    _POOL_MANAGER,
)
from app.config import get_settings
from app.providers.jimeng.errors import PoolConfigError
from app.providers.jimeng.store import AccountStore
from app.providers.jimeng.token_manager import TokenManager

router = APIRouter(prefix="/v1/jimeng", tags=["jimeng"])


class CreateJimengAccountRequest(BaseModel):
    sessionid: str = Field(..., description="即梦账号 sessionid")
    description: str = Field("", description="可选：备注/描述")
    enabled: bool = Field(True, description="是否启用")


class UpdateJimengAccountRequest(BaseModel):
    sessionid: Optional[str] = Field(None, description="可选：更新 sessionid")
    description: Optional[str] = Field(None, description="可选：更新描述")
    enabled: Optional[bool] = Field(None, description="可选：更新启用状态")


@lru_cache(maxsize=1)
def _get_store() -> AccountStore:
    s = get_settings()
    return AccountStore(s.jimeng_pool_db_path)


@router.get("/accounts")
def list_accounts(include_disabled: bool = True) -> Dict[str, Any]:
    store = _get_store()
    accounts = store.list_accounts(include_disabled=include_disabled)
    return {
        "data": {
            "list": [acc.to_safe_dict() for acc in accounts],
            "total": len(accounts),
        }
    }


@router.post("/accounts")
def create_account(req: CreateJimengAccountRequest) -> Dict[str, Any]:
    store = _get_store()
    try:
        record = store.create_account(sessionid=req.sessionid, description=req.description)
        if req.enabled is not True:
            record = store.update_account(account_id=record.id, enabled=bool(req.enabled))
        return {"data": record.to_safe_dict()}
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@router.patch("/accounts/{account_id}")
def update_account(account_id: int, req: UpdateJimengAccountRequest) -> Dict[str, Any]:
    store = _get_store()
    try:
        record = store.update_account(
            account_id=account_id,
            sessionid=req.sessionid,
            description=req.description,
            enabled=req.enabled,
        )
        return {"data": record.to_safe_dict()}
    except ValueError as exc:
        msg = str(exc)
        status = 404 if "不存在" in msg else 400
        raise HTTPException(status_code=status, detail=msg) from exc


@router.get("/pool/accounts")
def list_pool_accounts() -> Dict[str, Any]:
    # 复用适配器的号池管理器（与 /v1/generate 使用同一实例）
    adapter = JimengAdapter()
    try:
        pool = _POOL_MANAGER.get_pool(adapter.cfg)
    except PoolConfigError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    return {"data": [asdict(view) for view in pool.accounts_view()]}


@router.post("/pool/reload")
def reload_pool() -> Dict[str, Any]:
    # 与适配器共享同一 manager：确保 reload 对生成逻辑生效
    try:
        ok = _POOL_MANAGER.invalidate_if_idle()
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    if not ok:
        raise HTTPException(status_code=409, detail="号池忙碌，请稍后重试（等待当前任务完成）")
    return {"data": {"reloaded": True}}


@router.post("/accounts/{account_id}/credit")
def refresh_credit(account_id: int) -> Dict[str, Any]:
    record = _get_store().get_account(account_id)
    if not record:
        raise HTTPException(status_code=404, detail="账号不存在")

    account_config = copy.deepcopy(JimengAdapter().cfg.base_config)
    account_config["accounts"] = [{"sessionid": record.sessionid, "description": record.description}]
    total = TokenManager(account_config).get_credit()
    total_credit = None if not total else total.get("total_credit")
    return {"data": {"account_id": int(account_id), "total_credit": total_credit, "detail": total}}


@router.post("/accounts/{account_id}/daily-credit")
def receive_daily_credit(account_id: int) -> Dict[str, Any]:
    record = _get_store().get_account(account_id)
    if not record:
        raise HTTPException(status_code=404, detail="账号不存在")

    account_config = copy.deepcopy(JimengAdapter().cfg.base_config)
    account_config["accounts"] = [{"sessionid": record.sessionid, "description": record.description}]
    total_credit = TokenManager(account_config).receive_daily_credit()
    return {"data": {"account_id": int(account_id), "total_credit": total_credit}}
