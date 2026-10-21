use std::collections::VecDeque;

/// `FairQueue` hands out one item at a time, rotating across submitters so a
/// single busy caller cannot monopolise the single execution slot.
///
/// The remote-desktop workflow execution is deliberately serialised (one
 /// workflow at a time per process, because it owns the screen, focus and
/// input state). That makes the *order* the fairness question: a plain FIFO
/// lets whoever submits first - or submits fastest - push every other
 /// submitter's work behind an unbounded number of their own runs.
///
/// The rule is round-robin over submitters: each round serves at most one
/// item per submitter, in submission order inside that submitter. A
/// submitter that is already running still counts as "served this round",
/// so it cannot queue-jump past the others.
#[derive(Debug)]
pub struct FairQueue<T> {
    /// Submitter keys in the order they first appeared.
    order: Vec<String>,
    /// Per-submitter FIFO of waiting items.
    lanes: std::collections::HashMap<String, VecDeque<T>>,
    /// Index of the next submitter to serve.
    cursor: usize,
    len: usize,
}

impl<T> FairQueue<T> {
    pub fn new() -> Self {
        Self {
            order: Vec::new(),
            lanes: std::collections::HashMap::new(),
            cursor: 0,
            len: 0,
        }
    }

    pub fn len(&self) -> usize {
        self.len
    }

    /// Convenience for callers and tests that only need "is there anything".
    /// The production worker always pops and matches on `Option`, so this is
    /// not on its hot path.
    #[allow(dead_code)]
    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    /// Number of items currently waiting for one submitter.
    pub fn waiting_for(&self, key: &str) -> usize {
        self.lanes.get(key).map(VecDeque::len).unwrap_or(0)
    }

    /// Appends an item to its submitter's lane, creating the lane on first
    /// use so every submitter gets exactly one turn per round.
    pub fn push(&mut self, key: &str, item: T) {
        if !self.lanes.contains_key(key) {
            self.order.push(key.to_string());
        }
        self.lanes
            .entry(key.to_string())
            .or_default()
            .push_back(item);
        self.len += 1;
    }

    /// Removes and returns the next item, rotating to the submitter after
    /// `cursor`. Returns `None` when nothing is waiting.
    pub fn pop(&mut self) -> Option<T> {
        if self.order.is_empty() {
            return None;
        }
        // The cursor points at the next submitter to serve, so the search
        // starts there. Bounded by the number of lanes: every lane is
        // visited at most once.
        let lanes = self.order.len();
        for step in 0..lanes {
            let index = (self.cursor + step) % lanes;
            let key = &self.order[index];
            let Some(lane) = self.lanes.get_mut(key) else {
                continue;
            };
            let Some(item) = lane.pop_front() else {
                continue;
            };
            self.len -= 1;
            self.cursor = index;
            if lane.is_empty() {
                self.lanes.remove(key);
                self.order.remove(index);
                // Removing a lane shifts every later index down by one, so
                // the cursor has to follow the lane it just served.
                if self.cursor > index {
                    self.cursor -= 1;
                }
                if self.order.is_empty() {
                    self.cursor = 0;
                } else {
                    self.cursor %= self.order.len();
                }
            } else {
                // A lane that still has work is served at most once per
                // round, so the cursor must move past it before returning.
                // Without this the same lane would be picked again on the
                // next pop and the rotation would collapse into FIFO for
                // whichever submitter happened to be first.
                self.cursor = (index + 1) % self.order.len();
            }
            return Some(item);
        }
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn empty_queue_yields_nothing() {
        let mut queue = FairQueue::<u32>::new();
        assert!(queue.is_empty());
        assert_eq!(queue.pop(), None);
    }

    // The core guarantee: with one submitter holding three items and another
    // holding one, the single item is served second rather than fourth.
    #[test]
    fn one_busy_submitter_cannot_monopolise_the_slot() {
        let mut queue = FairQueue::new();
        queue.push("busy", 1);
        queue.push("busy", 2);
        queue.push("quiet", 3);
        queue.push("busy", 4);

        assert_eq!(queue.len(), 4);
        assert_eq!(queue.pop(), Some(1));
        assert_eq!(queue.pop(), Some(3));
        assert_eq!(queue.pop(), Some(2));
        assert_eq!(queue.pop(), Some(4));
        assert!(queue.is_empty());
        assert_eq!(queue.pop(), None);
    }

    // A submitter that only ever has one outstanding item still gets served
    // every round, and the rotation does not drop or duplicate anything.
    #[test]
    fn round_robin_covers_every_submitter_before_repeating() {
        let mut queue = FairQueue::new();
        for submitter in ["a", "b", "c"] {
            queue.push(submitter, submitter.to_string());
        }

        let mut served = Vec::new();
        while let Some(item) = queue.pop() {
            served.push(item);
        }
        assert_eq!(served, vec!["a", "b", "c"]);
    }

    // Lanes must be reclaimed once drained, otherwise a submitter that
    // disappears keeps a permanent turn in the rotation.
    #[test]
    fn drained_lanes_are_reclaimed() {
        let mut queue = FairQueue::new();
        queue.push("gone", 1);
        queue.push("stays", 2);

        assert_eq!(queue.pop(), Some(1));
        assert_eq!(queue.len(), 1);
        queue.push("stays", 3);

        assert_eq!(queue.pop(), Some(2));
        assert_eq!(queue.pop(), Some(3));
        assert!(queue.is_empty());
    }

    // Interleaved submission from many submitters must not lose or reorder
    // items inside a single lane.
    #[test]
    fn per_submitter_order_is_preserved() {
        let mut queue = FairQueue::new();
        for round in 0..3 {
            queue.push("alpha", format!("a{round}"));
            queue.push("beta", format!("b{round}"));
        }

        let mut alpha = Vec::new();
        let mut beta = Vec::new();
        while let Some(item) = queue.pop() {
            if item.starts_with('a') {
                alpha.push(item);
            } else {
                beta.push(item);
            }
        }
        assert_eq!(alpha, vec!["a0", "a1", "a2"]);
        assert_eq!(beta, vec!["b0", "b1", "b2"]);
    }

    // The rotation must keep working after a lane is reclaimed mid-stream:
    // a submitter that runs out and then comes back later is a new lane, not
    // a stale slot in the old rotation.
    #[test]
    fn a_returning_submitter_gets_a_fresh_turn() {
        let mut queue = FairQueue::new();
        queue.push("one", 1);
        queue.push("two", 2);

        assert_eq!(queue.pop(), Some(1));
        assert_eq!(queue.pop(), Some(2));
        assert!(queue.is_empty());

        queue.push("one", 3);
        queue.push("two", 4);
        assert_eq!(queue.pop(), Some(3));
        assert_eq!(queue.pop(), Some(4));
        assert!(queue.is_empty());
    }

    // Stress the index bookkeeping that the simpler cases cannot reach: many
    // lanes, each drained at a different time, must never lose an item.
    #[test]
    fn many_lanes_drained_at_different_times_lose_nothing() {
        let mut queue = FairQueue::new();
        let mut expected = 0;
        for submitter in 0..5 {
            for round in 0..(submitter + 1) {
                queue.push(&format!("lane{submitter}"), (submitter, round));
                expected += 1;
            }
        }

        let mut seen = 0;
        while queue.pop().is_some() {
            seen += 1;
        }
        assert_eq!(seen, expected);
        assert!(queue.is_empty());
    }

    // Per-submitter depth is what the caller-facing capacity check reads, so
    // it has to count only that submitter's outstanding items.
    #[test]
    fn waiting_for_counts_only_that_submitter() {
        let mut queue = FairQueue::new();
        assert_eq!(queue.waiting_for("a"), 0);

        queue.push("a", 1);
        queue.push("a", 2);
        queue.push("b", 3);
        assert_eq!(queue.waiting_for("a"), 2);
        assert_eq!(queue.waiting_for("b"), 1);
        assert_eq!(queue.waiting_for("never-seen"), 0);

        assert_eq!(queue.pop(), Some(1));
        assert_eq!(queue.waiting_for("a"), 1);
        assert_eq!(queue.pop(), Some(3));
        assert_eq!(queue.waiting_for("b"), 0);
        assert_eq!(queue.waiting_for("a"), 1);
    }
}
