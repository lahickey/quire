//! A tiny inventory: items, stock levels, and reorder rules.

use std::collections::{BTreeMap, HashMap};
use std::fmt::{self, Display};
use std::str::FromStr;

const REORDER_FLOOR: u32 = 10;
static WAREHOUSE: &str = "north";

/// What can go wrong when stock moves.
#[derive(Debug, Clone, PartialEq)]
pub enum StockError {
    Unknown(String),
    Short { sku: String, wanted: u32, held: u32 },
    Parse(std::num::ParseIntError),
}

impl Display for StockError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unknown(sku) => write!(f, "no such item: {sku}"),
            Self::Short { sku, wanted, held } => {
                write!(f, "{sku}: wanted {wanted} but only {held} held")
            }
            Self::Parse(e) => write!(f, "bad number: {e}"),
        }
    }
}

impl std::error::Error for StockError {}

impl From<std::num::ParseIntError> for StockError {
    fn from(e: std::num::ParseIntError) -> Self {
        StockError::Parse(e)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub struct Pence(pub u64);

#[derive(Debug, Clone)]
pub struct Item<'a> {
    pub sku: &'a str,
    pub held: u32,
    pub price: Pence,
    pub tags: Vec<&'a str>,
}

pub trait Priced {
    fn value(&self) -> Pence;

    fn is_cheap(&self) -> bool {
        self.value() < Pence(1_000)
    }
}

impl Priced for Item<'_> {
    fn value(&self) -> Pence {
        Pence(self.price.0 * u64::from(self.held))
    }
}

#[derive(Default)]
pub struct Inventory<'a> {
    items: HashMap<&'a str, Item<'a>>,
    log: Vec<String>,
}

impl<'a> Inventory<'a> {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn add(&mut self, item: Item<'a>) -> Option<Item<'a>> {
        self.log.push(format!("add {} x{}", item.sku, item.held));
        self.items.insert(item.sku, item)
    }

    /* Take `count` of `sku` out of stock, or say why not. */
    pub fn take(&mut self, sku: &str, count: u32) -> Result<u32, StockError> {
        let item = self
            .items
            .get_mut(sku)
            .ok_or_else(|| StockError::Unknown(sku.to_owned()))?;
        if item.held < count {
            return Err(StockError::Short { sku: sku.into(), wanted: count, held: item.held });
        }
        item.held -= count;
        self.log.push(format!("take {sku} x{count}"));
        Ok(item.held)
    }

    pub fn to_reorder(&self) -> impl Iterator<Item = &Item<'a>> + '_ {
        self.items.values().filter(|i| i.held <= REORDER_FLOOR)
    }

    pub fn by_tag(&self) -> BTreeMap<&str, Vec<&str>> {
        let mut out: BTreeMap<&str, Vec<&str>> = BTreeMap::new();
        for item in self.items.values() {
            for tag in &item.tags {
                out.entry(tag).or_default().push(item.sku);
            }
        }
        out
    }
}

fn parse_line(line: &str) -> Result<(&str, u32), StockError> {
    let (sku, count) = line.split_once(':').unwrap_or((line, "1"));
    Ok((sku.trim(), u32::from_str(count.trim())?))
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut inv = Inventory::new();
    inv.add(Item { sku: "bolt-m4", held: 250, price: Pence(12), tags: vec!["steel", "small"] });
    inv.add(Item { sku: "hinge", held: 8, price: Pence(4_50), tags: vec!["steel"] });

    let orders = r#"bolt-m4: 40
hinge: 12
washer"#;
    for (n, line) in orders.lines().enumerate() {
        match parse_line(line).and_then(|(sku, count)| inv.take(sku, count)) {
            Ok(left) => println!("{:>2}. ok, {left} left", n + 1),
            Err(e @ StockError::Short { .. }) => eprintln!("{:>2}. short: {e}", n + 1),
            Err(e) => eprintln!("{:>2}. {e}", n + 1),
        }
    }

    let low: Vec<_> = inv.to_reorder().map(|i| i.sku).collect();
    let total: u64 = inv.items.values().map(|i| i.value().0).sum();
    let ratio = total as f64 / 1.0e2;
    println!("{WAREHOUSE}: reorder {low:?}, worth £{ratio:.2}, byte {:#04x}, char {:?}", b'\n', '\u{1F600}');
    assert!(inv.by_tag().contains_key("steel"), "tags are kept");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_a_count() {
        assert_eq!(parse_line("hinge: 3").unwrap(), ("hinge", 3));
    }
}
