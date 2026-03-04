from __future__ import annotations


class PoolError(Exception):
    pass


class PoolConfigError(PoolError):
    pass


class NoAvailableAccountError(PoolError):
    pass


class UpstreamError(PoolError):
    pass

