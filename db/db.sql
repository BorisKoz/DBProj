CREATE EXTENSION IF NOT EXISTS citext;

-- tables for value storing
CREATE UNLOGGED TABLE users (
    Nickname citext PRIMARY KEY NOT NULL,
    FullName citext NOT NULL,
    About TEXT,
    Email citext UNIQUE
);
CREATE UNLOGGED TABLE Forum (
    Title TEXT,
    Usr citext,
    Slug citext PRIMARY KEY,
    Posts BIGINT DEFAULT 0,
    Threads INT DEFAULT 0,
    FOREIGN KEY (Usr) REFERENCES users(Nickname)
);
CREATE UNLOGGED TABLE Threads (
    Id BIGSERIAL PRIMARY KEY,
    Title TEXT NOT NULL,
    Forum citext NOT NULL,
    Message TEXT,
    Author citext,
    Votes BIGINT DEFAULT 0,
    Slug citext,
    Created TIMESTAMP WITH TIME ZONE DEFAULT now(),
    FOREIGN KEY (Author) REFERENCES users(Nickname),
    FOREIGN KEY (Forum) REFERENCES  Forum(Slug)
);
-- as slug is optional, and pgx cannot read null strings
CREATE UNIQUE INDEX idx_unq_thread_slug ON Threads(Slug) WHERE Slug <> '';

-- deleted fkey parent as the referencing is done by path and 0 violates constraints
CREATE UNLOGGED TABLE Posts (
    Id BIGSERIAL PRIMARY KEY ,
    Parent BIGINT DEFAULT 0,
    Author citext,
    Message TEXT,
    IsEdited BOOLEAN,
    Forum citext,
    Thread BIGINT,
    CREATED TIMESTAMP WITH TIME ZONE DEFAULT now(),
    FOREIGN KEY (Author) REFERENCES users(Nickname),
    FOREIGN KEY (Forum) REFERENCES Forum(Slug),
    FOREIGN KEY (Thread) REFERENCES Threads(Id),
    treeOrder BIGINT[]
);
CREATE UNLOGGED TABLE Votes (
    Nickname citext,
    Voice INT,
    IdThread BIGINT,
    IdVote BIGSERIAL PRIMARY KEY ,
    FOREIGN KEY (Nickname) REFERENCES users(Nickname),
    FOREIGN KEY (IdThread) REFERENCES Threads(Id),
    UNIQUE (Nickname, IdThread)
);
-- redundancy adding table, however shortens getting all users on forum (otherwise - comparing two selects)
CREATE UNLOGGED TABLE forumUsers (
    Nickname citext,
    Slug citext,
    UNIQUE (Nickname, Slug),
    FOREIGN KEY (Nickname) REFERENCES users(Nickname),
    FOREIGN KEY (Slug) REFERENCES Forum(Slug)
);
-- Next lie functions that fill in the counting tables -
-- instead of costly count(*) use denormalized fields
-- implementing count as trigger increments

-- adding users to forumUsers
CREATE OR REPLACE FUNCTION forumAddUser() RETURNS TRIGGER AS
    $forumAddUser$
    BEGIN
        INSERT INTO forumUsers (nickname, slug)  VALUES  (NEW.Author, NEW.Forum) ON CONFLICT DO NOTHING;
        RETURN NEW;
    end;
    $forumAddUser$
LANGUAGE plpgsql;
CREATE TRIGGER newThreadCreated AFTER INSERT
    ON Threads FOR EACH ROW
    EXECUTE PROCEDURE forumAddUser();
CREATE TRIGGER newPostCreated AFTER INSERT
    ON Posts FOR EACH ROW
    EXECUTE PROCEDURE forumAddUser();

-- counting threads of forum
CREATE OR REPLACE FUNCTION forumAddThread() RETURNS TRIGGER AS
    $forumAddThread$
    BEGIN
        UPDATE Forum SET Threads=Threads+1 WHERE LOWER(Slug) = LOWER(NEW.Forum);
        RETURN NEW;
    end;
    $forumAddThread$
LANGUAGE plpgsql;
CREATE TRIGGER newThreadCreated1 AFTER INSERT
    ON Threads FOR EACH ROW
    EXECUTE PROCEDURE forumAddThread();

-- hierarchy of posts + counting forum posts
CREATE OR REPLACE FUNCTION forumCheckPost() RETURNS TRIGGER AS
    $forumCheckPost$
    DECLARE
        parentThread BIGINT;
        parentTreeOrder BIGINT[];
    BEGIN
        IF EXISTS(SELECT Nickname FROM users WHERE LOWER(Nickname) = LOWER(NEW.Author)) = false THEN
            RAISE EXCEPTION 'ERR FOREIGN KEY VIOLATION' USING ERRCODE ='23503';
        end if;
        IF EXISTS(SELECT ID FROM Threads WHERE Threads.Id = NEW.Thread) = false THEN
            RAISE EXCEPTION 'ERR FOREIGN KEY VIOLATION' USING ERRCODE ='23503';
        end if;
        -- check post for parent of thread
        IF (NEW.Parent <> 0) THEN
            SELECT Thread from Posts WHERE Id = NEW.Parent INTO parentThread;
            IF NOT FOUND OR parentThread != NEW.thread THEN
                RAISE EXCEPTION 'DIFFERENT PARENT' USING ERRCODE = '23505';
                    -- this block raises the UniqueViolation ERRCODE, which leads to 409 on response
            end if;
        end if;
        -- update post count and paths
        UPDATE Forum SET Posts=Posts+1 WHERE LOWER(Slug) = LOWER(NEW.Forum);
        IF (NEW.Parent = 0) THEN
            NEW.treeOrder = NEW.treeOrder || NEW.Id;
        ELSE
            SELECT treeOrder FROM Posts WHERE id = NEW.Parent INTO parentTreeOrder;
            NEW.treeOrder = NEW.treeOrder || parentTreeOrder || NEW.Id;
        end if;
    RETURN NEW;
    end;
    $forumCheckPost$
LANGUAGE plpgsql;
CREATE TRIGGER newPostToAdd BEFORE INSERT
    ON Posts FOR EACH ROW
    EXECUTE PROCEDURE forumCheckPost();

-- votes, as it can be +-1, no need to get previous vote values
CREATE OR REPLACE FUNCTION threadAddVote() RETURNS TRIGGER AS
    $threadAddVote$
    BEGIN
        UPDATE Threads SET Votes=Votes+NEW.Voice WHERE Id = NEW.IdThread;
        RETURN NEW;
    end;
    $threadAddVote$
LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION threadChangeVote() RETURNS TRIGGER AS
    $threadAddVote$
    BEGIN
        IF OLD.Voice <> NEW.Voice THEN
            UPDATE Threads SET Votes=Votes+NEW.Voice*2 WHERE Id = NEW.IdThread;
        end if;
        return NEW;
    end;
    $threadAddVote$
LANGUAGE plpgsql;
CREATE TRIGGER newVote AFTER INSERT
    ON Votes FOR EACH ROW
    EXECUTE PROCEDURE threadAddVote();
CREATE TRIGGER changeVote AFTER UPDATE
    ON Votes FOR EACH ROW
    EXECUTE PROCEDURE threadChangeVote();

--indexes
--user -index nickame and email, lower and normal
CREATE INDEX usersNicknameIndex ON users (Nickname);
CREATE INDEX usersEmailIndex ON users (Email);
--forum - index slug
CREATE INDEX forumSlugIndex ON Forum (Forum.Slug);
--threads
CREATE INDEX threadSlugIndex ON Threads (Slug);
CREATE INDEX threadForumLowerIndex ON Threads (Forum);
CREATE INDEX threadCreatedIndex ON Threads (Created);
--posts indexing thread, path and parent, also order by's
CREATE INDEX postThreadIndex ON Posts (Thread);
CREATE INDEX postOrderIndex ON Posts ((Posts.treeOrder));
CREATE INDEX postOrder1Index ON Posts ((Posts.treeOrder[1]));
CREATE INDEX postThreadParentIdIndex ON Posts (Thread, (Parent), Id);
CREATE INDEX postCreatedIndex ON Posts (Created);
--orders
CREATE INDEX postOrderOrder1OrderIdIndex ON Posts ((Posts.treeOrder[1]), (Posts.treeOrder), id);
CREATE INDEX postOrderOrder1ThreadIndex ON Posts ((Posts.treeOrder[1]), Thread);
--votes
CREATE INDEX voteNicknameIndex ON votes (lower(Nickname), IdThread, Voice);
--forumUser
CREATE INDEX forumUsersNicknameIndex ON forumUsers (Nickname);
CREATE INDEX forumUsersForumIndex ON forumUsers (Slug);