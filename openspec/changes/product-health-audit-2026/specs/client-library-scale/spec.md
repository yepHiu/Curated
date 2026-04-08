## ADDED Requirements

### Requirement: Large-library browse SHALL remain bounded on the client

When the Web library adapter loads movies for browsing, the system SHALL NOT grow unbounded in-memory movie arrays without a documented cap or progressive loading strategy.

#### Scenario: Prefetch cap is enforced

- **WHEN** the library contains more movies than the configured maximum prefetch limit
- **THEN** the client SHALL not load more than that limit into memory for the default browse list, OR the product SHALL surface explicit truncation to the user (e.g. total vs loaded count)

#### Scenario: Virtual list does not imply unbounded data

- **WHEN** the library grid uses virtual scrolling for rendering
- **THEN** the data source strategy SHALL be reviewed so that rendering optimizations are not undermined by loading an entire catalog into a single reactive array
