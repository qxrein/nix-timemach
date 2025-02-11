use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct Generation {
    pub id: String,
    pub timestamp: DateTime<Utc>,
    pub description: Option<String>,
    pub profiles: Vec<String>,
    pub current: bool,
}
