"""Bounded email lookup within the caller's past travel contacts."""
def distance(a, b):
    previous = list(range(len(b) + 1))
    for i, left in enumerate(a):
        current = [i + 1]
        for j, right in enumerate(b):
            current.append(min(current[-1] + 1, previous[j + 1] + 1, previous[j] + (left != right)))
        previous = current
    return previous[-1]


def rank_emails(records, query):
    query = query.strip().lower()
    result = []
    for record in records:
        email = record.get('email', '').lower()
        score = 1000
        if email == query:
            score = 0
        elif query in email:
            score = 1
        elif len(query) >= 4:
            target = email if '@' in query else email.split('@')[0]
            typo = distance(query, target)
            if typo <= 2:
                score = typo + 2
        if score < 1000:
            result.append((score, email, record))
    return [record for _, _, record in sorted(result, key=lambda item: (item[0], item[1]))[:20]]
