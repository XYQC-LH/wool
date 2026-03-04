from __future__ import annotations

import copy
import threading
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

from .api_client import ApiClient
from .credit_policy import DEFAULT_IMAGES_PER_REQUEST, estimate_required_credit
from .token_manager import TokenManager

from .errors import NoAvailableAccountError, PoolConfigError, UpstreamError


@dataclass
class AccountView:
    id: int
    description: str
    enabled: bool
    unhealthy_until: float
    last_error: Optional[str]
    cached_credit_total: Optional[int]
    cached_credit_at: float


class AccountRuntime:
    def __init__(self, account_id: int, sessionid: str, description: str, base_config: Dict[str, Any]):
        self.id = int(account_id)
        self.enabled = True
        self.sessionid = str(sessionid or "").strip()
        self.description = str(description or "").strip() or f"Account {self.id}"

        if not self.sessionid:
            raise PoolConfigError("sessionid 不能为空")

        account_data = {"sessionid": self.sessionid, "description": self.description}

        account_config = copy.deepcopy(base_config)
        account_config["accounts"] = [account_data]
        self.config = account_config

        self.token_manager = TokenManager(account_config)
        self.api_client = ApiClient(self.token_manager, account_config)

        self.lock = threading.Lock()

        self.unhealthy_until = 0.0
        self.last_error: Optional[str] = None

        self.cached_credit_total: Optional[int] = None
        self.cached_credit_at: float = 0.0

    def view(self) -> AccountView:
        return AccountView(
            id=self.id,
            description=self.description,
            enabled=self.enabled,
            unhealthy_until=float(self.unhealthy_until or 0.0),
            last_error=self.last_error,
            cached_credit_total=self.cached_credit_total,
            cached_credit_at=float(self.cached_credit_at or 0.0),
        )


class JimengPool:
    def __init__(
        self,
        base_config: Dict[str, Any],
        accounts: List[Dict[str, Any]],
        *,
        credit_cache_seconds: int = 60,
        account_cooldown_seconds: int = 60,
    ):
        if not isinstance(accounts, list):
            raise PoolConfigError("accounts 必须为 list")

        self.base_config = base_config
        self.credit_cache_seconds = max(int(credit_cache_seconds), 0)
        self.account_cooldown_seconds = max(int(account_cooldown_seconds), 0)

        self._accounts: List[AccountRuntime] = []
        self._accounts_by_id: Dict[int, AccountRuntime] = {}
        for row in accounts:
            if not isinstance(row, dict):
                continue

            try:
                account_id = int(row.get("id"))
            except Exception:
                continue

            sessionid = str(row.get("sessionid") or "").strip()
            if not sessionid:
                continue

            enabled = row.get("enabled", True)
            if enabled is not None and not bool(enabled):
                continue

            runtime = AccountRuntime(
                account_id=account_id,
                sessionid=sessionid,
                description=str(row.get("description") or ""),
                base_config=base_config,
            )
            self._accounts.append(runtime)
            self._accounts_by_id[runtime.id] = runtime

        if not self._accounts:
            raise PoolConfigError("未找到可用账号（数据库为空或全部禁用）")

        self._accounts.sort(key=lambda item: item.id)

        self._rr_lock = threading.Lock()
        self._rr_next = 0

        timeout_cfg = base_config.get("timeout") if isinstance(base_config.get("timeout"), dict) else {}
        self.max_wait_time = int(timeout_cfg.get("max_wait_time") or 180)
        self.check_interval = int(timeout_cfg.get("check_interval") or 10)
        if self.check_interval <= 0:
            self.check_interval = 10
        if self.max_wait_time <= 0:
            self.max_wait_time = 180

    def accounts_view(self) -> List[AccountView]:
        return [account.view() for account in self._accounts]

    def is_idle(self) -> bool:
        for account in self._accounts:
            acquired = account.lock.acquire(blocking=False)
            if not acquired:
                return False
            account.lock.release()
        return True

    def _next_start_index(self) -> int:
        with self._rr_lock:
            start = self._rr_next
            self._rr_next = (self._rr_next + 1) % len(self._accounts)
            return start

    def _refresh_credit_locked(self, account: AccountRuntime) -> Optional[int]:
        credit_info = account.token_manager.get_credit()
        total = None if not credit_info else credit_info.get("total_credit")
        try:
            total_int = int(total) if total is not None else None
        except Exception:
            total_int = None

        account.cached_credit_total = total_int
        account.cached_credit_at = time.time()
        return total_int

    def refresh_credit(self, account_id: int) -> Optional[int]:
        account = self._get_account(account_id)
        with account.lock:
            return self._refresh_credit_locked(account)

    def receive_daily_credit(self, account_id: int) -> Optional[int]:
        account = self._get_account(account_id)
        with account.lock:
            total = account.token_manager.receive_daily_credit()
            try:
                total_int = int(total) if total is not None else None
            except Exception:
                total_int = None
            if total_int is not None:
                account.cached_credit_total = total_int
                account.cached_credit_at = time.time()
            return total_int

    def _get_account(self, account_id: int) -> AccountRuntime:
        try:
            account_id_int = int(account_id)
        except Exception as exc:
            raise PoolConfigError("account_id 非法") from exc

        runtime = self._accounts_by_id.get(account_id_int)
        if not runtime:
            raise PoolConfigError("账号不存在或已禁用")
        return runtime

    def reserve_for_model(self, model: str, *, preferred_account_id: Optional[int] = None) -> AccountRuntime:
        required_credit = estimate_required_credit(str(model), images_per_request=DEFAULT_IMAGES_PER_REQUEST)
        return self.reserve_account(required_credit=required_credit, preferred_account_id=preferred_account_id)

    def reserve_account(self, *, required_credit: Optional[int], preferred_account_id: Optional[int]) -> AccountRuntime:
        now = time.time()
        candidates: List[AccountRuntime]
        if preferred_account_id is not None:
            try:
                only_id = int(preferred_account_id)
            except Exception as exc:
                raise PoolConfigError("preferred_account 非法") from exc

            account = self._accounts_by_id.get(only_id)
            if not account:
                raise PoolConfigError("preferred_account 不存在或已禁用")
            candidates = [account]
        else:
            start = self._next_start_index()
            candidates = [
                self._accounts[(start + offset) % len(self._accounts)] for offset in range(len(self._accounts))
            ]

        last_error: Optional[str] = None
        for account in candidates:
            if account.unhealthy_until and account.unhealthy_until > now:
                continue

            acquired = account.lock.acquire(blocking=False)
            if not acquired:
                continue

            try:
                if required_credit is not None:
                    total = account.cached_credit_total
                    is_stale = (now - (account.cached_credit_at or 0.0)) > float(self.credit_cache_seconds)
                    if total is None or is_stale:
                        total = self._refresh_credit_locked(account)
                    if total is None or total < required_credit:
                        last_error = f"账号积分不足：{total} < {required_credit}"
                        account.lock.release()
                        continue
                return account
            except Exception:
                account.lock.release()
                raise

        raise NoAvailableAccountError(last_error or "没有可用账号（忙/熔断/积分不足）")

    def _poll_images_locked(self, account: AccountRuntime, history_id: str) -> List[str]:
        start = time.monotonic()
        while True:
            urls = account.api_client._get_generated_images_by_history_id(history_id)
            if urls:
                return list(urls)

            if (time.monotonic() - start) >= float(self.max_wait_time):
                raise UpstreamError(f"生成超时（history_id={history_id}）")

            time.sleep(float(self.check_interval))

    def generate_t2i_with_account(
        self,
        account: AccountRuntime,
        *,
        prompt: str,
        model: str,
        ratio: str,
        seed: int = -1,
    ) -> Dict[str, Any]:
        try:
            submit_resp = account.api_client.submit_t2i(prompt=prompt, model=model, ratio=ratio, seed=seed)
            if not submit_resp or submit_resp.get("error"):
                raise UpstreamError(str(submit_resp))

            history_id = submit_resp.get("history_id") or submit_resp.get("history_record_id")
            if not history_id:
                raise UpstreamError(f"missing history_id: {submit_resp}")

            urls = self._poll_images_locked(account, str(history_id))
            return {
                "urls": urls,
                "history_id": str(history_id),
                "queue_message": submit_resp.get("queue_message"),
                "account": {"id": account.id, "description": account.description},
            }
        except Exception as exc:
            account.last_error = str(exc)
            account.unhealthy_until = time.time() + float(self.account_cooldown_seconds)
            raise
        finally:
            try:
                account.lock.release()
            except RuntimeError:
                pass

    def generate_t2i(
        self,
        *,
        prompt: str,
        model: str,
        ratio: str,
        seed: int = -1,
        preferred_account_id: Optional[int] = None,
    ) -> Dict[str, Any]:
        account = self.reserve_for_model(model, preferred_account_id=preferred_account_id)
        return self.generate_t2i_with_account(account, prompt=prompt, model=model, ratio=ratio, seed=seed)

    def generate_i2i_reference_with_account(
        self,
        account: AccountRuntime,
        *,
        prompt: str,
        model: str,
        ratio: str,
        reference_image_paths: List[str],
    ) -> Dict[str, Any]:
        try:
            submit_resp = account.api_client.upload_image_and_generate_with_reference(
                image_paths=list(reference_image_paths),
                prompt=prompt,
                model=model,
                ratio=ratio,
            )
            if not submit_resp or submit_resp.get("error"):
                raise UpstreamError(str(submit_resp))

            history_id = submit_resp.get("history_id") or submit_resp.get("history_record_id")
            if not history_id:
                raise UpstreamError(f"missing history_id: {submit_resp}")

            urls = submit_resp.get("urls")
            if not urls:
                urls = self._poll_images_locked(account, str(history_id))

            return {
                "urls": list(urls),
                "history_id": str(history_id),
                "queue_message": submit_resp.get("queue_message"),
                "account": {"id": account.id, "description": account.description},
            }
        except Exception as exc:
            account.last_error = str(exc)
            account.unhealthy_until = time.time() + float(self.account_cooldown_seconds)
            raise
        finally:
            try:
                account.lock.release()
            except RuntimeError:
                pass

    def generate_i2i_reference(
        self,
        *,
        prompt: str,
        model: str,
        ratio: str,
        reference_image_paths: List[str],
        preferred_account_id: Optional[int] = None,
    ) -> Dict[str, Any]:
        account = self.reserve_for_model(model, preferred_account_id=preferred_account_id)
        return self.generate_i2i_reference_with_account(
            account,
            prompt=prompt,
            model=model,
            ratio=ratio,
            reference_image_paths=reference_image_paths,
        )
