# Monban Models

This package contains structures, datatypes and functions/interfaces used across multiple packages within Monban. Any
data that is to be shared across packages should be created in this package.

## Interfaces
Most datatypes have interfaces implemented to make working with them easier and abstract the underlying data structures.
As those might change the interfaces stay consistent and reduce changes when some under-the-hood things get changed.
