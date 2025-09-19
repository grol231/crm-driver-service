"""Logging utilities for tests."""
import logging
import structlog
import sys
from typing import Any, Dict
from config import get_test_config

config = get_test_config()


def setup_logger() -> structlog.stdlib.BoundLogger:
    """Setup structured logger for tests."""
    
    # Configure structlog
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="ISO"),
            structlog.dev.ConsoleRenderer() if config.log_format == "console" else structlog.processors.JSONRenderer(),
            structlog.processors.add_log_level,
            structlog.processors.StackInfoRenderer(),
        ],
        wrapper_class=structlog.stdlib.BoundLogger,
        logger_factory=structlog.stdlib.LoggerFactory(),
        context_class=dict,
        cache_logger_on_first_use=True,
    )
    
    # Configure standard library logging
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=getattr(logging, config.log_level.upper()),
    )
    
    return structlog.get_logger("integration_tests")


def get_logger(name: str = "integration_tests") -> structlog.stdlib.BoundLogger:
    """Get logger instance."""
    return structlog.get_logger(name)