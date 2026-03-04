from __future__ import annotations

import os
import sqlite3
import threading
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional


def _mask_secret(value: str) -> str:
    raw = str(value or "")
    if not raw:
        return "********"
    if len(raw) <= 8:
        return "********"
    return f"{raw[:4]}****{raw[-4:]}"


@dataclass(frozen=True)
class AccountRecord:
    id: int
    sessionid: str
    description: str
    enabled: bool
    created_at: int
    updated_at: int

    def to_safe_dict(self) -> Dict[str, Any]:
        return {
            "id": int(self.id),
            "description": str(self.description or ""),
            "enabled": bool(self.enabled),
            "sessionid_masked": _mask_secret(self.sessionid),
            "created_at": int(self.created_at),
            "updated_at": int(self.updated_at),
        }


class AccountStore:
    def __init__(self, db_path: str):
        self.db_path = str(db_path or "").strip() or "jimeng_pool.db"
        db_dir = os.path.dirname(self.db_path)
        if db_dir:
            os.makedirs(db_dir, exist_ok=True)

        self._lock = threading.Lock()
        self._conn = sqlite3.connect(self.db_path, check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._init_schema()

    def _init_schema(self) -> None:
        with self._lock:
            cur = self._conn.cursor()
            cur.execute("PRAGMA foreign_keys=ON;")
            try:
                cur.execute("PRAGMA journal_mode=WAL;")
            except Exception:
                pass

            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS jimeng_accounts (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  sessionid TEXT NOT NULL UNIQUE,
                  description TEXT NOT NULL DEFAULT '',
                  enabled INTEGER NOT NULL DEFAULT 1,
                  created_at INTEGER NOT NULL,
                  updated_at INTEGER NOT NULL
                );
                """
            )
            cur.execute("CREATE INDEX IF NOT EXISTS idx_jimeng_accounts_enabled ON jimeng_accounts(enabled);")
            self._conn.commit()

    def count_accounts(self) -> int:
        with self._lock:
            row = self._conn.execute("SELECT COUNT(1) AS c FROM jimeng_accounts;").fetchone()
            return int(row["c"]) if row else 0

    def enabled_signature(self) -> tuple[int, int]:
        """
        返回启用账号的轻量签名，用于判断是否需要重载：
        (enabled_count, max_updated_at)
        """
        with self._lock:
            row = self._conn.execute(
                "SELECT COUNT(1) AS c, COALESCE(MAX(updated_at), 0) AS m FROM jimeng_accounts WHERE enabled = 1;"
            ).fetchone()
            if not row:
                return 0, 0
            try:
                return int(row["c"] or 0), int(row["m"] or 0)
            except Exception:
                return 0, 0

    def list_accounts(self, *, include_disabled: bool = True) -> List[AccountRecord]:
        sql = "SELECT id, sessionid, description, enabled, created_at, updated_at FROM jimeng_accounts"
        params: tuple[Any, ...] = ()
        if not include_disabled:
            sql += " WHERE enabled = 1"
        sql += " ORDER BY id ASC"

        with self._lock:
            rows = self._conn.execute(sql, params).fetchall()
            return [self._row_to_record(row) for row in rows]

    def get_account(self, account_id: int) -> Optional[AccountRecord]:
        with self._lock:
            row = self._conn.execute(
                "SELECT id, sessionid, description, enabled, created_at, updated_at FROM jimeng_accounts WHERE id = ?",
                (int(account_id),),
            ).fetchone()
            return self._row_to_record(row) if row else None

    def create_account(self, *, sessionid: str, description: str = "") -> AccountRecord:
        sessionid = str(sessionid or "").strip()
        if not sessionid:
            raise ValueError("sessionid 不能为空")
        description = str(description or "").strip()
        now = int(time.time())

        with self._lock:
            try:
                cur = self._conn.execute(
                    """
                    INSERT INTO jimeng_accounts(sessionid, description, enabled, created_at, updated_at)
                    VALUES (?, ?, 1, ?, ?)
                    """,
                    (sessionid, description, now, now),
                )
                self._conn.commit()
                account_id = int(cur.lastrowid)
            except sqlite3.IntegrityError as exc:
                raise ValueError("sessionid 已存在") from exc

        record = self.get_account(account_id)
        if not record:
            raise RuntimeError("创建账号失败")
        return record

    def update_account(
        self,
        *,
        account_id: int,
        sessionid: Optional[str] = None,
        description: Optional[str] = None,
        enabled: Optional[bool] = None,
    ) -> AccountRecord:
        updates: List[str] = []
        params: List[Any] = []

        if sessionid is not None:
            sessionid = str(sessionid or "").strip()
            if not sessionid:
                raise ValueError("sessionid 不能为空")
            updates.append("sessionid = ?")
            params.append(sessionid)

        if description is not None:
            updates.append("description = ?")
            params.append(str(description or "").strip())

        if enabled is not None:
            updates.append("enabled = ?")
            params.append(1 if bool(enabled) else 0)

        if not updates:
            record = self.get_account(int(account_id))
            if not record:
                raise ValueError("账号不存在")
            return record

        updates.append("updated_at = ?")
        params.append(int(time.time()))
        params.append(int(account_id))

        with self._lock:
            try:
                cur = self._conn.execute(
                    f"UPDATE jimeng_accounts SET {', '.join(updates)} WHERE id = ?",
                    tuple(params),
                )
                self._conn.commit()
            except sqlite3.IntegrityError as exc:
                raise ValueError("sessionid 已存在") from exc

            if cur.rowcount <= 0:
                raise ValueError("账号不存在")

        record = self.get_account(int(account_id))
        if not record:
            raise ValueError("账号不存在")
        return record

    @staticmethod
    def _row_to_record(row: sqlite3.Row) -> AccountRecord:
        return AccountRecord(
            id=int(row["id"]),
            sessionid=str(row["sessionid"] or ""),
            description=str(row["description"] or ""),
            enabled=bool(int(row["enabled"] or 0)),
            created_at=int(row["created_at"] or 0),
            updated_at=int(row["updated_at"] or 0),
        )
